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
          - heading "Connexion" [level=1] [ref=e8]
          - paragraph [ref=e9]: Connectez-vous a votre compte
        - generic [ref=e10]:
          - generic [ref=e11]:
            - generic [ref=e12]: Email
            - textbox "Email" [active] [ref=e13]:
              - /placeholder: vous@exemple.com
              - text: not-an-email
          - generic [ref=e14]:
            - generic [ref=e15]: Mot de passe
            - textbox "Mot de passe" [ref=e16]:
              - /placeholder: Votre mot de passe
              - text: TestPass1234!
            - link "Mot de passe oublie ?" [ref=e18] [cursor=pointer]:
              - /url: /forgot-password
          - button "Se connecter" [ref=e19]
          - paragraph [ref=e20]:
            - text: Pas encore de compte ?
            - link "Creer un compte" [ref=e21] [cursor=pointer]:
              - /url: /register
  - button "Open Next.js Dev Tools" [ref=e27] [cursor=pointer]:
    - img [ref=e28]
  - alert [ref=e31]
```