# Herald — Plan d'actions v0.1 → Public Release

> Généré le 12 février 2026 après audit complet du repo
> Objectif : premier push public GitHub, projet crédible et sécurisé

---

## État actuel

- **MVP fonctionnel** : 9 outils MCP, OAuth 2.1 + PKCE, SQLite, exécution async, isolation Git
- **6 940 lignes Go**, 45 fichiers source, 13 fichiers tests, 5 dépendances
- **6/6 vulnérabilités CRITICAL corrigées**
- **0 TODO/FIXME**, `go vet` clean, tous tests passent
- **Pas de remote Git**, pas de Makefile, pas de Dockerfile
- **Couverture inégale** : task 91%, store 85%, mais executor 19%, handlers 20%

---

## Phase 1 — Fondations manquantes (pré-requis push public)

### 1.1 Makefile

Priorité : **BLOQUANT**

Premier fichier que les contributeurs regardent. Doit couvrir :

- `make build` — binaire unique CGO_ENABLED=0
- `make test` — tests avec race detector
- `make test-cover` — couverture avec rapport HTML
- `make lint` — golangci-lint
- `make dev` — hot reload (air ou watchexec)
- `make clean` — nettoyage
- `make install` — installation locale
- `make release` — goreleaser snapshot

### 1.2 .goreleaser.yml

Priorité : **HAUTE**

Multi-plateforme (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64). Checksums SHA256. Archives tar.gz. Pas de publication automatique pour l'instant.

### 1.3 Dockerfile

Priorité : **HAUTE**

Multi-stage build :
- Stage 1 : Go 1.26, build statique
- Stage 2 : scratch ou distroless, copier le binaire
- Exposer port 8420
- Volume pour config et data
- Note dans le README : le binaire direct est recommandé (accès filesystem + claude CLI)

### 1.4 .github/workflows/

Priorité : **HAUTE**

Vérifier/compléter les workflows existants (commit `cdcac19`) :
- `ci.yml` : build + test + lint sur push/PR
- `release.yml` : goreleaser sur tag v*
- `security.yml` : govulncheck + golangci-lint security

### 1.5 Remote Git + premier push

Priorité : **BLOQUANT**

- Créer le repo `kolapsis/herald` sur GitHub
- `git remote add origin`
- Vérifier `.gitignore` (pas de secrets, pas de binaires, pas de .db)
- Push main
- Protéger la branche main (require PR pour la suite)

---

## Phase 2 — Sécurité HIGH (avant publication)

Les 5 trouvailles HIGH de l'audit sécurité. Chacune est une PR indépendante.

### 2.1 H1 — Symlink traversal dans read_file

Priorité : **CRITIQUE pour la publication**

`read_file` valide le chemin avec `filepath.Abs` + prefix check, mais ne résout pas les symlinks. Un symlink dans le projet peut pointer vers `/etc/shadow`.

Fix : `filepath.EvalSymlinks` avant la validation du prefix. Tester avec un symlink qui pointe hors du projet.

### 2.2 H2 — Prompt size illimité

Priorité : **HAUTE**

`start_task` accepte n'importe quelle taille de prompt. Un prompt de 100MB crashe Herald (OOM) ou génère un coût API astronomique.

Fix : Ajouter `max_prompt_size` dans ExecutionConfig (défaut 100KB). Valider dans le handler avant d'écrire le fichier prompt. HTTP 400 si dépassé.

### 2.3 H3 — Output mémoire non borné

Priorité : **HAUTE**

Le stream parser accumule l'output complet en mémoire (`task.AppendOutput`). Une tâche avec un output de 500MB = OOM.

Fix : Ring buffer ou écriture sur disque avec limite configurable. Garder les N dernières lignes en mémoire pour `check_task`, écrire le reste sur disque pour `get_result`.

### 2.4 H4 — Permissions base de données

Priorité : **MOYENNE**

Le fichier SQLite est créé avec les permissions par défaut (0644). Devrait être 0600 (lecture/écriture owner uniquement).

Fix : `os.OpenFile` avec 0600, ou `os.Chmod` après création. Vérifier aussi le répertoire parent (0700).

### 2.5 H5 — Security headers manquants

Priorité : **MOYENNE**

Les réponses HTTP ne contiennent pas les headers de sécurité standards. Traefik peut les ajouter, mais Herald devrait les inclure par défaut.

Fix : Middleware qui ajoute `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: strict-origin-when-cross-origin`, `Cache-Control: no-store` sur les endpoints sensibles.

---

## Phase 3 — Couverture de tests critique

Objectif : passer au-dessus de 60% sur TOUS les packages avant publication.

### 3.1 Executor — stream parser (19% → 70%+)

Priorité : **CRITIQUE**

C'est le code qui parse l'output de Claude Code. Chaque bug ici = tâche perdue ou résultat corrompu.

Tests à écrire :
- Parsing de chaque type d'événement (system/init, assistant/text, assistant/tool_use, result/success, result/error)
- Lignes JSON malformées (ne doit pas crasher)
- Buffer overflow (lignes très longues)
- Stream interrompu (EOF inattendu)
- Événements dans le désordre
- Extraction du session_id
- Extraction du coût et des turns

### 3.2 MCP Handlers (20% → 70%+)

Priorité : **HAUTE**

6 handlers sur 9 n'ont aucun test. Chaque handler doit avoir au minimum :
- Test du happy path (requête valide → réponse attendue)
- Test des paramètres manquants/invalides
- Test des erreurs (tâche introuvable, projet inexistant)

Handlers non testés : `check_task`, `get_result`, `cancel_task`, `get_diff`, `get_logs`, `list_tasks`

### 3.3 Config loader (59% → 80%+)

Priorité : **MOYENNE**

Tests à ajouter :
- Chargement YAML valide avec tous les champs
- Chargement YAML avec champs manquants (defaults)
- Override par variables d'environnement
- Validation des valeurs (port range, chemins existants, TTL parseable)
- Fichier inexistant → erreur claire

### 3.4 Git operations (66% → 80%+)

Priorité : **MOYENNE**

Tests à ajouter :
- Stash sur working tree dirty
- Création de branche sur repo avec des branches existantes
- Diff entre branches
- Gestion d'un repo sans commits

---

## Phase 4 — Qualité de code et polish

### 4.1 Nettoyage des packages vides

Priorité : **HAUTE**

Supprimer ou marquer clairement les packages stub qui n'ont pas d'implémentation. Un contributeur ne doit pas être confus.

Options :
- **Supprimer** `internal/api/`, `internal/dashboard/`, `internal/pipeline/`, `internal/template/` → les recréer quand on les implémente
- **OU** Garder avec un `doc.go` qui explique "planned for v0.3, not yet implemented"

Recommandation : supprimer. C'est du bruit. Git garde l'historique.

### 4.2 Vérifier la cohérence herald.example.yaml ↔ config.go

Priorité : **MOYENNE**

L'exemple YAML doit refléter exactement les champs du struct Config. Après les ajouts de sécurité (redirect_uris, rate_limit, etc.), vérifier que l'exemple est à jour et documenté.

### 4.3 LICENSE

Priorité : **BLOQUANT**

Vérifier qu'un fichier LICENSE (MIT) existe à la racine. Pas vu dans le listing.

### 4.4 .gitignore

Priorité : **HAUTE**

Vérifier/compléter :
```
# Binaires
herald
bin/
*.exe

# Data
*.db
*.db-journal
*.db-wal

# Config avec secrets
herald.yaml
!configs/herald.example.yaml

# IDE
.idea/
.vscode/
*.swp

# OS
.DS_Store
Thumbs.db

# Go
vendor/
```

Note : le pattern `herald` dans .gitignore peut matcher le binaire ET le répertoire. Vérifier que `cmd/herald/main.go` n'est pas ignoré par erreur (c'était un problème lors du fix C5).

### 4.5 go.mod cleanup

Priorité : **BASSE**

`go mod tidy` pour nettoyer les dépendances inutilisées. Vérifier que `go.sum` est committé.

---

## Phase 5 — Documentation pour le lancement

### 5.1 README.md — vérification finale

Priorité : **HAUTE**

Le README existe (commit `546b2cd`). Vérifier :
- Quick Start fonctionne réellement (copier-coller les commandes)
- Les liens internes fonctionnent
- La section Configuration reflète la config actuelle
- Screenshots ou GIF de demo (optionnel mais fort impact)

### 5.2 CONTRIBUTING.md

Priorité : **MOYENNE**

Court et efficace :
- Comment builder le projet
- Comment lancer les tests
- Convention de commits (conventional commits)
- Convention de branches (feature/, fix/, docs/)
- Process PR

### 5.3 SECURITY.md

Priorité : **HAUTE**

Politique de divulgation responsable. Adresse email ou moyen de reporter une vulnérabilité. Standard pour tout projet de sécurité.

### 5.4 Nettoyage des docs de planification

Priorité : **MOYENNE**

Les fichiers `herald-ceo-plan.md` et `herald-cto-spec.md` sont des documents internes. Options :
- Les déplacer dans `docs/internal/` (accessibles mais pas à la racine)
- Les retirer du repo public (garder en local)

Recommandation : déplacer dans `docs/internal/`. La transparence est cohérente avec le "build in public".

---

## Phase 6 — Post-publication (v0.2+)

Ces éléments ne bloquent PAS la publication mais sont sur la roadmap.

### 6.1 Memory v0.2
Branche `feat/memory-v0.2` avec scaffolding prêt. Contexte partagé Chat ↔ Code.

### 6.2 Notifications webhook + SSE
Compléter `internal/notify/` (ntfy existe, ajouter webhook et SSE).

### 6.3 Dashboard web
SPA embarquée via `go:embed`, SSE temps réel, HTML/CSS/JS vanilla.

### 6.4 REST API
`/api/v1/*` pour les intégrations programmatiques (curl, n8n, etc.)

### 6.5 Templates et pipelines
Moteur de templates YAML, pipelines multi-étapes avec conditions.

### 6.6 CLI complète
`herald check`, `herald init`, `herald hash-password`, `herald generate-token`

---

## Ordre d'exécution recommandé

```
Phase 1.3  LICENSE                          5 min
Phase 1.1  Makefile                        15 min
Phase 1.4  .gitignore vérification          5 min
Phase 4.1  Supprimer packages vides        10 min
Phase 4.5  go mod tidy                      2 min
Phase 2.1  Fix H1 symlink traversal        15 min
Phase 2.2  Fix H2 prompt size              10 min
Phase 2.3  Fix H3 output mémoire           20 min
Phase 2.4  Fix H4 DB permissions            5 min
Phase 2.5  Fix H5 security headers         10 min
Phase 3.1  Tests executor stream parser     30 min
Phase 3.2  Tests MCP handlers              30 min
Phase 3.3  Tests config loader             15 min
Phase 3.4  Tests git operations            15 min
Phase 1.2  .goreleaser.yml                 10 min
Phase 1.3  Dockerfile                      10 min
Phase 1.4  GitHub Actions vérification     15 min
Phase 5.1  README vérification finale      10 min
Phase 5.2  CONTRIBUTING.md                 10 min
Phase 5.3  SECURITY.md                     10 min
Phase 5.4  Nettoyage docs internes          5 min
Phase 4.2  herald.example.yaml sync         5 min
Phase 1.5  Remote Git + push              10 min
                                    ─────────────
                                    Total : ~5h
```

Tout est faisable via Herald en une session. Les temps estimés sont pour l'exécution par Claude Code, pas pour un humain.

---

## Critères de "done" pour le push public

- [ ] Tous les tests passent avec race detector
- [ ] Couverture > 60% sur tous les packages avec du code
- [ ] 0 finding CRITICAL ou HIGH non traité
- [ ] Makefile fonctionnel (build, test, lint, clean)
- [ ] LICENSE MIT à la racine
- [ ] README avec Quick Start qui fonctionne
- [ ] SECURITY.md
- [ ] .gitignore propre (pas de secrets, pas de binaires)
- [ ] go vet clean
- [ ] golangci-lint clean (ou warnings documentés)
- [ ] Pas de packages vides sans explication
- [ ] herald.example.yaml à jour
