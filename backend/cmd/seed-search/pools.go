package main

// pools.go holds every curated list the realistic-dataset generator
// draws from. Keeping them in a separate file makes it obvious which
// constants to tweak when the sample dataset needs refreshing.

// city + coordinates distribution (top 30 EU + NA markets). The
// latitude / longitude allow the geo filter to return non-empty
// results from realistic probes.
var citiesPool = []struct {
	City, Country string
	Lat, Lng      float64
}{
	{"Paris", "FR", 48.8566, 2.3522},
	{"Lyon", "FR", 45.7640, 4.8357},
	{"Marseille", "FR", 43.2965, 5.3698},
	{"Bordeaux", "FR", 44.8378, -0.5792},
	{"Toulouse", "FR", 43.6047, 1.4442},
	{"Nantes", "FR", 47.2184, -1.5536},
	{"Lille", "FR", 50.6292, 3.0573},
	{"Strasbourg", "FR", 48.5734, 7.7521},
	{"Nice", "FR", 43.7102, 7.2620},
	{"Montpellier", "FR", 43.6110, 3.8767},
	{"Berlin", "DE", 52.5200, 13.4050},
	{"Munich", "DE", 48.1351, 11.5820},
	{"Hamburg", "DE", 53.5511, 9.9937},
	{"London", "GB", 51.5074, -0.1278},
	{"Manchester", "GB", 53.4808, -2.2426},
	{"Edinburgh", "GB", 55.9533, -3.1883},
	{"Amsterdam", "NL", 52.3676, 4.9041},
	{"Rotterdam", "NL", 51.9244, 4.4777},
	{"Madrid", "ES", 40.4168, -3.7038},
	{"Barcelona", "ES", 41.3851, 2.1734},
	{"Lisbon", "PT", 38.7223, -9.1393},
	{"Porto", "PT", 41.1579, -8.6291},
	{"Rome", "IT", 41.9028, 12.4964},
	{"Milan", "IT", 45.4642, 9.1900},
	{"Dublin", "IE", 53.3498, -6.2603},
	{"Brussels", "BE", 50.8503, 4.3517},
	{"Zurich", "CH", 47.3769, 8.5417},
	{"New York", "US", 40.7128, -74.0060},
	{"San Francisco", "US", 37.7749, -122.4194},
	{"Montreal", "CA", 45.5017, -73.5673},
}

// firstNamesFR + firstNamesEN give the display-name generator enough
// spread to avoid duplicate full names within 500 profiles.
var firstNamesFR = []string{
	"Camille", "Alexandre", "Sophie", "Julien", "Manon", "Thomas",
	"Lucie", "Nicolas", "Chloé", "Antoine", "Pauline", "Maxime",
	"Clara", "Romain", "Emma", "Pierre", "Léa", "Baptiste",
	"Juliette", "Hugo", "Marine", "Victor", "Inès", "Louis",
}

var firstNamesEN = []string{
	"James", "Olivia", "Liam", "Emma", "William", "Sophia",
	"Benjamin", "Ava", "Noah", "Isabella", "Ethan", "Charlotte",
	"Lucas", "Amelia", "Mason", "Harper", "Logan", "Evelyn",
}

var lastNamesFR = []string{
	"Martin", "Bernard", "Dubois", "Thomas", "Robert", "Petit",
	"Durand", "Leroy", "Moreau", "Simon", "Laurent", "Lefebvre",
	"Michel", "Garcia", "David", "Bertrand", "Roux", "Vincent",
	"Fournier", "Morel", "Girard", "Andre", "Lefevre", "Mercier",
}

var lastNamesEN = []string{
	"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia",
	"Miller", "Davis", "Rodriguez", "Martinez", "Hernandez", "Lopez",
	"Gonzalez", "Wilson", "Anderson", "Thomas", "Taylor", "Moore",
}

// titlesFreelance covers the persona's typical job boards. 30+ variants
// so list views feel like a real marketplace, not a test harness.
var titlesFreelance = []string{
	"Senior React Developer",
	"Full-stack Engineer — Next.js + Node.js",
	"Product Designer — SaaS B2B",
	"Lead iOS Developer (Swift / SwiftUI)",
	"Android Engineer — Kotlin + Jetpack Compose",
	"Flutter Developer — cross-platform apps",
	"DevOps Engineer — Kubernetes + Terraform",
	"Data Engineer — Airflow + dbt",
	"Machine Learning Engineer — LangChain + LLM",
	"Backend Engineer — Go + PostgreSQL",
	"Technical Lead — FinTech",
	"Frontend Architect — TypeScript + Vue",
	"UX Designer — mobile-first",
	"UI Designer — design systems",
	"Motion Designer — After Effects + Lottie",
	"3D Artist — Blender + Three.js",
	"Copywriter FR — SaaS B2B",
	"Content Strategist EN — Growth",
	"SEO Consultant — technical audits",
	"Growth Marketer — paid acquisition",
	"CRM Specialist — HubSpot + Salesforce",
	"Email Automation Expert",
	"Product Manager — API platforms",
	"Scrum Master — agile coaching",
	"QA Engineer — Cypress + Playwright",
	"Security Engineer — pentesting + SOC 2",
	"Cloud Architect — AWS + multi-region",
	"Site Reliability Engineer",
	"Database Administrator — PostgreSQL + Redis",
	"Blockchain Developer — Solidity + Ethers.js",
}

// titlesAgency — agency names skew descriptive and often carry a
// city / specialty marker.
var titlesAgency = []string{
	"Agence webdesign Paris — branding + sites e-commerce",
	"Studio React / Next.js — SaaS + FinTech",
	"Agence SEO & contenu FR / EN",
	"Studio mobile iOS + Android",
	"Studio branding + motion design",
	"Agence CRM & marketing automation",
	"Studio UX research + design system",
	"Agence développement Shopify + headless",
	"Agence WordPress / WooCommerce sur mesure",
	"Studio DevOps — SRE + Cloud Architecture",
	"Agence data engineering & BI",
	"Studio Blockchain — DeFi + NFT",
	"Agence croissance B2B — ABM + inbound",
	"Studio produit — MVP + design sprint",
	"Agence IA — chatbots + LLM sur mesure",
}

// titlesReferrer — apporteurs d'affaires carry a sector-oriented pitch.
var titlesReferrer = []string{
	"Apporteur d'affaires SaaS B2B",
	"Business Referrer — FinTech + InsurTech",
	"Apporteur d'affaires e-commerce",
	"Référencement grands comptes CAC 40",
	"Apporteur d'affaires industries (PME + ETI)",
	"Business Development — RetailTech",
	"Apporteur d'affaires santé + MedTech",
	"Connector — Silicon Valley ↔ Europe",
	"Apporteur d'affaires LegalTech",
	"Business Referrer — AgriTech + FoodTech",
	"Apporteur d'affaires mobilité + automobile",
	"Réseau grands groupes énergie + utilities",
	"Apporteur d'affaires immobilier + PropTech",
	"Business Referrer — education + EdTech",
	"Apporteur d'affaires industrie 4.0",
}

// skillPool holds ~100 tech + non-tech skills following Pareto.
// First 20 are the "hot" skills every dev mentions, then a long tail.
var skillPool = []string{
	"React", "TypeScript", "Node.js", "Go", "Python",
	"Next.js", "Vue", "PostgreSQL", "Docker", "Kubernetes",
	"AWS", "GCP", "Redis", "GraphQL", "MongoDB",
	"Figma", "Tailwind", "Rust", "Ruby", "PHP",
	"Java", "Kotlin", "Swift", "Flutter", "Dart",
	"LangChain", "LLM", "OpenAI", "Anthropic", "Pinecone",
	"Terraform", "Ansible", "CI/CD", "GitHub Actions", "GitLab CI",
	"Cypress", "Playwright", "Jest", "Vitest", "Pytest",
	"Django", "FastAPI", "Flask", "Laravel", "Symfony",
	"Express", "NestJS", "Spring Boot", ".NET", "Rails",
	"Stripe", "PayPal", "Adyen", "Mollie", "WebRTC",
	"Twilio", "Mailchimp", "HubSpot", "Salesforce", "Zendesk",
	"Photoshop", "Illustrator", "Sketch", "Webflow", "Framer",
	"Blender", "Cinema 4D", "After Effects", "Premiere Pro", "DaVinci",
	"SEO", "SEM", "Ahrefs", "SEMrush", "Google Analytics",
	"Copywriting FR", "Copywriting EN", "Ghostwriting", "Social Media", "Community Management",
	"Product Management", "Scrum", "Agile Coaching", "OKR", "Roadmapping",
	"Machine Learning", "Deep Learning", "PyTorch", "TensorFlow", "Scikit-learn",
	"Data Engineering", "Airflow", "dbt", "Snowflake", "BigQuery",
	"Kafka", "RabbitMQ", "Elasticsearch", "Typesense", "Algolia",
	"Cybersecurity", "Pentesting", "SOC 2", "ISO 27001", "GDPR compliance",
}

// expertisePool maps to the domain/expertise keys exposed by the
// backend catalog. Keep in sync with EXPERTISE_DOMAIN_KEYS when the
// catalog ships.
var expertisePool = []string{
	"dev-frontend", "dev-backend", "dev-mobile", "dev-fullstack",
	"data-ml", "data-engineering", "design-ux", "design-ui", "design-motion",
	"marketing-growth", "marketing-seo", "marketing-content",
	"ops-devops", "ops-sre", "ops-security",
	"product-management", "product-discovery",
	"sales-bizdev", "finance-accounting", "legal-compliance",
}

// aboutSnippets is the pool the bio generator draws from. Each entry
// is a single sentence; the seeder joins two or three to form 2-3
// sentence bios. Variants reference the skill / persona passed in.
var aboutFreelanceSnippets = []string{
	"Ten years shipping production software for startups and Fortune 500 teams.",
	"Focused on pragmatic architecture: small services, fast feedback loops, happy developers.",
	"Open-source contributor with a track record in OSS communities.",
	"Speak at tech conferences regularly (Paris Web, dotJS, KubeCon).",
	"Available for long-term engagements (3-12 months) and advisory retainers.",
	"Built several products from zero to multi-million-user scale.",
	"Strong bias toward simple, boring, battle-tested solutions.",
	"Comfortable owning the entire stack from infrastructure to end-user UX.",
	"Recently led a migration of a 50-engineer codebase to a monorepo.",
	"Coach + mentor — love raising the bar for the teams I work with.",
}

var aboutAgencySnippets = []string{
	"Full-service product studio: discovery, design, engineering, launch.",
	"Work with ambitious SaaS teams ready to scale from MVP to series B.",
	"Design-led — every engagement starts with a design sprint.",
	"Pragmatic timelines: 6-12 weeks for V1, then iterate.",
	"Based in Paris, serving clients across Europe and North America.",
	"Team of 12 seniors — no juniors, no offshore handoff.",
	"Deep expertise in headless commerce and B2B SaaS.",
	"Portfolio includes fintech, edtech, healthtech flagships.",
	"AWS + GCP certified partners.",
	"Ongoing support after launch — we don't disappear post-delivery.",
}

var aboutReferrerSnippets = []string{
	"Active network of 400+ decision-makers across the European tech scene.",
	"Worked 15 years in B2B sales before pivoting to independent referring.",
	"Specialised in complex enterprise introductions (> €100k contracts).",
	"Commission-based — aligned with your growth, not a retainer model.",
	"Pre-qualify every opportunity to avoid wasted calls.",
	"Warm intros only — every referral comes with context and rapport.",
	"Monthly check-ins with clients to keep the pipeline fresh.",
	"Track record: 60+ successful introductions in the past 24 months.",
	"Happy to sign NDAs and exclusivity clauses up-front.",
	"Focus sectors: SaaS, FinTech, eCommerce, MarTech.",
}
