#!/bin/bash
# Script de lancement de la session de nuit
# Usage: ./run-nuit.sh

cd /home/hassad/Documents/marketplaceServiceGo

echo "=== SESSION DE NUIT - $(date) ==="
echo "Log: /home/hassad/Documents/marketplaceServiceGo/nuit.log"
echo "L'agent va travailler toute la nuit. Tu peux fermer ce terminal."
echo ""

nohup claude -p \
  --dangerously-skip-permissions \
  --model opus \
  --effort max \
  "Tu es un agent autonome de nuit. Tu travailles TOUTE la nuit sans t'arreter.

IMPORTANT: Tu n'as AUCUNE limite de tours. Continue jusqu'a avoir tout fini.

Lis le fichier plannuit.md a la racine du projet pour ton plan de travail complet.
Lis aussi CLAUDE.md et backend/CLAUDE.md et web/CLAUDE.md et mobile/CLAUDE.md pour comprendre l'architecture.
Lis aussi bugacorriger.md, auditperf.md, auditsecurite.md, auditqualite.md pour le contexte.

RESUME DES REGLES CRITIQUES:
1. JAMAIS casser le build. Avant chaque commit: cd backend && go build ./... && cd ../web && npx tsc --noEmit
2. Tests dans les bons dossiers (Go: *_test.go a cote du source, Web: __tests__/ dans le feature, Flutter: test/ miroir de lib/)
3. Commits atomiques avec prefix test:
4. Max 600 lignes par fichier, max 50 lignes par fonction de test
5. Ne pas modifier le code source sauf si bug empeche compilation des tests
6. Si un bug est trouve, le documenter dans bugacorriger.md
7. Table-driven tests (Go), describe/it (TS), group/test (Dart)
8. Mocker les dependances externes, jamais la logique testee

WORKFLOW:
1. Lire plannuit.md et suivre les phases dans l'ordre (Phase 1 a 12)
2. Pour chaque phase: lire le source, ecrire les tests, lancer les tests, fix si erreur, commit
3. Apres Phase 12: entrer dans la boucle d'auto-increment (Phase 13)
   - Analyser le projet pour trouver du code non teste
   - Ecrire de nouveaux tests
   - Commit
   - Recommencer jusqu'a ce qu'il n'y ait plus rien a tester
4. Quand TOUT est fini (Phase finale dans plannuit.md):
   - Ecrire l'heure de fin et les stats dans plannuit.md
   - Commit final + push origin main
   - pkill -f code (fermer VS Code)
   - sleep 10 && systemctl poweroff (eteindre l'ordi, pas besoin de sudo)

NE T'ARRETE PAS. Continue phase apres phase. Si une phase est bloquee apres 3 tentatives, passe a la suivante et reviens plus tard. Tu as toute la nuit." \
  > nuit.log 2>&1 &

echo "PID: $!"
echo "Pour suivre: tail -f nuit.log"
