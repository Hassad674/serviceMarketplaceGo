# Page snapshot

```yaml
- generic [ref=e1]:
  - generic [ref=e2]:
    - banner [ref=e3]:
      - link "Marketplace Service" [ref=e4] [cursor=pointer]:
        - /url: /
    - main [ref=e5]:
      - generic [ref=e6]:
        - generic [ref=e7]:
          - heading "Mot de passe oublie" [level=1] [ref=e8]
          - paragraph [ref=e9]: Entrez votre email pour recevoir un lien de reinitialisation
        - generic [ref=e10]:
          - generic [ref=e11]:
            - generic [ref=e12]: Email
            - textbox "Email" [active] [ref=e13]:
              - /placeholder: vous@exemple.com
              - text: not-an-email
          - button "Envoyer le lien de reinitialisation" [ref=e14]
          - paragraph [ref=e15]:
            - link "Retour a la connexion" [ref=e16] [cursor=pointer]:
              - /url: /login
  - button "Open Next.js Dev Tools" [ref=e22] [cursor=pointer]:
    - img [ref=e23]
  - alert [ref=e26]
```