// Package seed owns the static, code-defined data loaded into the
// database on first run (or via the seed CLI). It is intentionally a
// plain Go package with no external imports so agents can edit it
// without triggering cross-layer dependency checks.
//
// This file (together with skills_part2.go) holds the curated skills
// catalog: ~500 widely-used tools, frameworks, techniques and domain
// terms that populate the "browse by expertise" panels on the public
// marketplace. Each entry is pre-normalized (SkillText equals
// skill.NormalizeSkillText(DisplayText)) and tagged with 1 to 3
// expertise keys from the frozen expertise catalog.
//
// Invariants (all enforced by skills_test.go):
//
//   - SkillText is unique across the whole slice (primary key in
//     skills_catalog).
//   - SkillText == skill.NormalizeSkillText(DisplayText) — the test
//     recomputes it rather than trusting the literal.
//   - Every ExpertiseKeys entry is in expertise.All.
//   - Every skill has at least one and at most three expertise keys.
//   - No expertise key is duplicated inside a single ExpertiseKeys.
//   - Slice length >= 400 and every expertise domain has >= 15 skills.
//
// The list is split across two files only because a single file would
// exceed the 600-line project limit. skillsPart1 here holds the first
// seven expertise sections (development through marketing_growth);
// skillsPart2 in skills_part2.go holds the remaining eight. The final
// CuratedSkills value is the concatenation of both, assembled at
// package init time by appendSkills.
//
// Adding a new skill: append one line at the bottom of the relevant
// section in whichever file contains that expertise. Use proper
// display casing ("Next.js", not "nextjs"), keep SkillText lowercase /
// trimmed / whitespace-collapsed, and tag with the narrowest correct
// set of expertise keys. When in doubt, a single expertise key is
// better than three.
package seed

// SeedSkill is one curated entry in the skills catalog seed. Loaded at
// backend startup (or via the seed CLI) into the skills_catalog table
// with is_curated = true.
type SeedSkill struct {
	SkillText     string   // normalized form — primary key
	DisplayText   string   // user-visible casing ("React", "Next.js")
	ExpertiseKeys []string // 1-3 values, all must be in expertise.All
}

// CuratedSkills is the curated catalog of skills the marketplace
// seeds on first run. It is the concatenation of skillsPart1 and
// skillsPart2, built once at package init time. Each skill_text must
// be the result of applying skill.NormalizeSkillText to the
// display_text. This invariant is checked by the test suite.
var CuratedSkills = appendSkills(skillsPart1, skillsPart2)

// appendSkills concatenates any number of skill fragments into a
// single slice. It returns a fresh slice so callers cannot mutate the
// underlying fragment storage by accident.
func appendSkills(parts ...[]SeedSkill) []SeedSkill {
	total := 0
	for _, p := range parts {
		total += len(p)
	}
	out := make([]SeedSkill, 0, total)
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// skillsPart1 holds the first seven expertise sections. Entries are
// ordered to match expertise.All. Cross-domain tagging happens inline
// via ExpertiseKeys — placement in a section is just for readability,
// the test suite counts distribution via ExpertiseKeys.
var skillsPart1 = []SeedSkill{
	// ---------------------------------------------------------------
	// development (target ~80)
	// ---------------------------------------------------------------
	{SkillText: "javascript", DisplayText: "JavaScript", ExpertiseKeys: []string{"development"}},
	{SkillText: "typescript", DisplayText: "TypeScript", ExpertiseKeys: []string{"development"}},
	{SkillText: "python", DisplayText: "Python", ExpertiseKeys: []string{"development", "data_ai_ml"}},
	{SkillText: "go", DisplayText: "Go", ExpertiseKeys: []string{"development"}},
	{SkillText: "rust", DisplayText: "Rust", ExpertiseKeys: []string{"development"}},
	{SkillText: "java", DisplayText: "Java", ExpertiseKeys: []string{"development"}},
	{SkillText: "kotlin", DisplayText: "Kotlin", ExpertiseKeys: []string{"development"}},
	{SkillText: "swift", DisplayText: "Swift", ExpertiseKeys: []string{"development"}},
	{SkillText: "c#", DisplayText: "C#", ExpertiseKeys: []string{"development"}},
	{SkillText: "c++", DisplayText: "C++", ExpertiseKeys: []string{"development"}},
	{SkillText: "php", DisplayText: "PHP", ExpertiseKeys: []string{"development"}},
	{SkillText: "ruby", DisplayText: "Ruby", ExpertiseKeys: []string{"development"}},
	{SkillText: "elixir", DisplayText: "Elixir", ExpertiseKeys: []string{"development"}},
	{SkillText: "scala", DisplayText: "Scala", ExpertiseKeys: []string{"development"}},
	{SkillText: "dart", DisplayText: "Dart", ExpertiseKeys: []string{"development"}},
	{SkillText: "zig", DisplayText: "Zig", ExpertiseKeys: []string{"development"}},
	{SkillText: "react", DisplayText: "React", ExpertiseKeys: []string{"development"}},
	{SkillText: "next.js", DisplayText: "Next.js", ExpertiseKeys: []string{"development"}},
	{SkillText: "vue.js", DisplayText: "Vue.js", ExpertiseKeys: []string{"development"}},
	{SkillText: "nuxt", DisplayText: "Nuxt", ExpertiseKeys: []string{"development"}},
	{SkillText: "svelte", DisplayText: "Svelte", ExpertiseKeys: []string{"development"}},
	{SkillText: "sveltekit", DisplayText: "SvelteKit", ExpertiseKeys: []string{"development"}},
	{SkillText: "astro", DisplayText: "Astro", ExpertiseKeys: []string{"development"}},
	{SkillText: "remix", DisplayText: "Remix", ExpertiseKeys: []string{"development"}},
	{SkillText: "solidjs", DisplayText: "SolidJS", ExpertiseKeys: []string{"development"}},
	{SkillText: "angular", DisplayText: "Angular", ExpertiseKeys: []string{"development"}},
	{SkillText: "qwik", DisplayText: "Qwik", ExpertiseKeys: []string{"development"}},
	{SkillText: "htmx", DisplayText: "HTMX", ExpertiseKeys: []string{"development"}},
	{SkillText: "alpine.js", DisplayText: "Alpine.js", ExpertiseKeys: []string{"development"}},
	{SkillText: "tailwind css", DisplayText: "Tailwind CSS", ExpertiseKeys: []string{"development"}},
	{SkillText: "shadcn ui", DisplayText: "Shadcn UI", ExpertiseKeys: []string{"development", "design_ui_ux"}},
	{SkillText: "radix ui", DisplayText: "Radix UI", ExpertiseKeys: []string{"development"}},
	{SkillText: "chakra ui", DisplayText: "Chakra UI", ExpertiseKeys: []string{"development"}},
	{SkillText: "material ui", DisplayText: "Material UI", ExpertiseKeys: []string{"development"}},
	{SkillText: "storybook", DisplayText: "Storybook", ExpertiseKeys: []string{"development", "design_ui_ux"}},
	{SkillText: "node.js", DisplayText: "Node.js", ExpertiseKeys: []string{"development"}},
	{SkillText: "deno", DisplayText: "Deno", ExpertiseKeys: []string{"development"}},
	{SkillText: "bun", DisplayText: "Bun", ExpertiseKeys: []string{"development"}},
	{SkillText: "express.js", DisplayText: "Express.js", ExpertiseKeys: []string{"development"}},
	{SkillText: "fastify", DisplayText: "Fastify", ExpertiseKeys: []string{"development"}},
	{SkillText: "nestjs", DisplayText: "NestJS", ExpertiseKeys: []string{"development"}},
	{SkillText: "hono", DisplayText: "Hono", ExpertiseKeys: []string{"development"}},
	{SkillText: "django", DisplayText: "Django", ExpertiseKeys: []string{"development"}},
	{SkillText: "flask", DisplayText: "Flask", ExpertiseKeys: []string{"development"}},
	{SkillText: "fastapi", DisplayText: "FastAPI", ExpertiseKeys: []string{"development"}},
	{SkillText: "ruby on rails", DisplayText: "Ruby on Rails", ExpertiseKeys: []string{"development"}},
	{SkillText: "laravel", DisplayText: "Laravel", ExpertiseKeys: []string{"development"}},
	{SkillText: "symfony", DisplayText: "Symfony", ExpertiseKeys: []string{"development"}},
	{SkillText: "spring boot", DisplayText: "Spring Boot", ExpertiseKeys: []string{"development"}},
	{SkillText: ".net", DisplayText: ".NET", ExpertiseKeys: []string{"development"}},
	{SkillText: "asp.net core", DisplayText: "ASP.NET Core", ExpertiseKeys: []string{"development"}},
	{SkillText: "chi", DisplayText: "Chi", ExpertiseKeys: []string{"development"}},
	{SkillText: "gin", DisplayText: "Gin", ExpertiseKeys: []string{"development"}},
	{SkillText: "phoenix", DisplayText: "Phoenix", ExpertiseKeys: []string{"development"}},
	{SkillText: "actix", DisplayText: "Actix", ExpertiseKeys: []string{"development"}},
	{SkillText: "axum", DisplayText: "Axum", ExpertiseKeys: []string{"development"}},
	{SkillText: "graphql", DisplayText: "GraphQL", ExpertiseKeys: []string{"development"}},
	{SkillText: "rest api", DisplayText: "REST API", ExpertiseKeys: []string{"development"}},
	{SkillText: "grpc", DisplayText: "gRPC", ExpertiseKeys: []string{"development"}},
	{SkillText: "websockets", DisplayText: "WebSockets", ExpertiseKeys: []string{"development"}},
	{SkillText: "webrtc", DisplayText: "WebRTC", ExpertiseKeys: []string{"development"}},
	{SkillText: "trpc", DisplayText: "tRPC", ExpertiseKeys: []string{"development"}},
	{SkillText: "prisma", DisplayText: "Prisma", ExpertiseKeys: []string{"development"}},
	{SkillText: "drizzle orm", DisplayText: "Drizzle ORM", ExpertiseKeys: []string{"development"}},
	{SkillText: "sqlalchemy", DisplayText: "SQLAlchemy", ExpertiseKeys: []string{"development"}},
	{SkillText: "postgresql", DisplayText: "PostgreSQL", ExpertiseKeys: []string{"development", "data_ai_ml"}},
	{SkillText: "mysql", DisplayText: "MySQL", ExpertiseKeys: []string{"development"}},
	{SkillText: "sqlite", DisplayText: "SQLite", ExpertiseKeys: []string{"development"}},
	{SkillText: "mongodb", DisplayText: "MongoDB", ExpertiseKeys: []string{"development"}},
	{SkillText: "redis", DisplayText: "Redis", ExpertiseKeys: []string{"development"}},
	{SkillText: "elasticsearch", DisplayText: "Elasticsearch", ExpertiseKeys: []string{"development", "data_ai_ml"}},
	{SkillText: "supabase", DisplayText: "Supabase", ExpertiseKeys: []string{"development"}},
	{SkillText: "firebase", DisplayText: "Firebase", ExpertiseKeys: []string{"development"}},
	{SkillText: "appwrite", DisplayText: "Appwrite", ExpertiseKeys: []string{"development"}},
	{SkillText: "react native", DisplayText: "React Native", ExpertiseKeys: []string{"development"}},
	{SkillText: "flutter", DisplayText: "Flutter", ExpertiseKeys: []string{"development"}},
	{SkillText: "expo", DisplayText: "Expo", ExpertiseKeys: []string{"development"}},
	{SkillText: "swiftui", DisplayText: "SwiftUI", ExpertiseKeys: []string{"development"}},
	{SkillText: "jetpack compose", DisplayText: "Jetpack Compose", ExpertiseKeys: []string{"development"}},
	{SkillText: "ionic", DisplayText: "Ionic", ExpertiseKeys: []string{"development"}},
	{SkillText: "electron", DisplayText: "Electron", ExpertiseKeys: []string{"development"}},
	{SkillText: "tauri", DisplayText: "Tauri", ExpertiseKeys: []string{"development"}},
	{SkillText: "unity", DisplayText: "Unity", ExpertiseKeys: []string{"development", "design_3d_animation"}},
	{SkillText: "unreal engine", DisplayText: "Unreal Engine", ExpertiseKeys: []string{"development", "design_3d_animation"}},
	{SkillText: "godot", DisplayText: "Godot", ExpertiseKeys: []string{"development", "design_3d_animation"}},
	{SkillText: "docker", DisplayText: "Docker", ExpertiseKeys: []string{"development"}},
	{SkillText: "kubernetes", DisplayText: "Kubernetes", ExpertiseKeys: []string{"development"}},
	{SkillText: "terraform", DisplayText: "Terraform", ExpertiseKeys: []string{"development"}},
	{SkillText: "pulumi", DisplayText: "Pulumi", ExpertiseKeys: []string{"development"}},
	{SkillText: "ansible", DisplayText: "Ansible", ExpertiseKeys: []string{"development"}},
	{SkillText: "github actions", DisplayText: "GitHub Actions", ExpertiseKeys: []string{"development"}},
	{SkillText: "gitlab ci", DisplayText: "GitLab CI", ExpertiseKeys: []string{"development"}},
	{SkillText: "jenkins", DisplayText: "Jenkins", ExpertiseKeys: []string{"development"}},
	{SkillText: "aws", DisplayText: "AWS", ExpertiseKeys: []string{"development"}},
	{SkillText: "google cloud platform", DisplayText: "Google Cloud Platform", ExpertiseKeys: []string{"development"}},
	{SkillText: "microsoft azure", DisplayText: "Microsoft Azure", ExpertiseKeys: []string{"development"}},
	{SkillText: "cloudflare", DisplayText: "Cloudflare", ExpertiseKeys: []string{"development"}},
	{SkillText: "vercel", DisplayText: "Vercel", ExpertiseKeys: []string{"development"}},
	{SkillText: "netlify", DisplayText: "Netlify", ExpertiseKeys: []string{"development"}},
	{SkillText: "railway", DisplayText: "Railway", ExpertiseKeys: []string{"development"}},
	{SkillText: "fly.io", DisplayText: "Fly.io", ExpertiseKeys: []string{"development"}},
	{SkillText: "linux", DisplayText: "Linux", ExpertiseKeys: []string{"development"}},
	{SkillText: "nginx", DisplayText: "Nginx", ExpertiseKeys: []string{"development"}},
	{SkillText: "git", DisplayText: "Git", ExpertiseKeys: []string{"development"}},
	{SkillText: "playwright", DisplayText: "Playwright", ExpertiseKeys: []string{"development"}},
	{SkillText: "cypress", DisplayText: "Cypress", ExpertiseKeys: []string{"development"}},
	{SkillText: "vitest", DisplayText: "Vitest", ExpertiseKeys: []string{"development"}},
	{SkillText: "jest", DisplayText: "Jest", ExpertiseKeys: []string{"development"}},
	{SkillText: "pytest", DisplayText: "pytest", ExpertiseKeys: []string{"development"}},
	{SkillText: "selenium", DisplayText: "Selenium", ExpertiseKeys: []string{"development"}},
	{SkillText: "solidity", DisplayText: "Solidity", ExpertiseKeys: []string{"development"}},
	{SkillText: "web3", DisplayText: "Web3", ExpertiseKeys: []string{"development"}},
	{SkillText: "wordpress", DisplayText: "WordPress", ExpertiseKeys: []string{"development", "marketing_growth"}},
	{SkillText: "shopify", DisplayText: "Shopify", ExpertiseKeys: []string{"development", "marketing_growth"}},
	{SkillText: "webflow", DisplayText: "Webflow", ExpertiseKeys: []string{"development", "design_ui_ux"}},
	{SkillText: "framer", DisplayText: "Framer", ExpertiseKeys: []string{"development", "design_ui_ux"}},
	{SkillText: "accessibility (wcag)", DisplayText: "Accessibility (WCAG)", ExpertiseKeys: []string{"development", "design_ui_ux"}},
	{SkillText: "web performance optimization", DisplayText: "Web Performance Optimization", ExpertiseKeys: []string{"development"}},

	// ---------------------------------------------------------------
	// data_ai_ml (target ~40)
	// ---------------------------------------------------------------
	{SkillText: "machine learning", DisplayText: "Machine Learning", ExpertiseKeys: []string{"data_ai_ml"}},
	{SkillText: "deep learning", DisplayText: "Deep Learning", ExpertiseKeys: []string{"data_ai_ml"}},
	{SkillText: "pytorch", DisplayText: "PyTorch", ExpertiseKeys: []string{"data_ai_ml"}},
	{SkillText: "tensorflow", DisplayText: "TensorFlow", ExpertiseKeys: []string{"data_ai_ml"}},
	{SkillText: "keras", DisplayText: "Keras", ExpertiseKeys: []string{"data_ai_ml"}},
	{SkillText: "scikit-learn", DisplayText: "scikit-learn", ExpertiseKeys: []string{"data_ai_ml"}},
	{SkillText: "pandas", DisplayText: "pandas", ExpertiseKeys: []string{"data_ai_ml"}},
	{SkillText: "numpy", DisplayText: "NumPy", ExpertiseKeys: []string{"data_ai_ml"}},
	{SkillText: "polars", DisplayText: "Polars", ExpertiseKeys: []string{"data_ai_ml"}},
	{SkillText: "jupyter", DisplayText: "Jupyter", ExpertiseKeys: []string{"data_ai_ml"}},
	{SkillText: "r", DisplayText: "R", ExpertiseKeys: []string{"data_ai_ml"}},
	{SkillText: "hugging face", DisplayText: "Hugging Face", ExpertiseKeys: []string{"data_ai_ml"}},
	{SkillText: "langchain", DisplayText: "LangChain", ExpertiseKeys: []string{"data_ai_ml", "development"}},
	{SkillText: "llamaindex", DisplayText: "LlamaIndex", ExpertiseKeys: []string{"data_ai_ml", "development"}},
	{SkillText: "openai api", DisplayText: "OpenAI API", ExpertiseKeys: []string{"data_ai_ml", "development"}},
	{SkillText: "claude api", DisplayText: "Claude API", ExpertiseKeys: []string{"data_ai_ml", "development"}},
	{SkillText: "mistral", DisplayText: "Mistral", ExpertiseKeys: []string{"data_ai_ml", "development"}},
	{SkillText: "llama", DisplayText: "Llama", ExpertiseKeys: []string{"data_ai_ml"}},
	{SkillText: "gemini api", DisplayText: "Gemini API", ExpertiseKeys: []string{"data_ai_ml", "development"}},
	{SkillText: "prompt engineering", DisplayText: "Prompt Engineering", ExpertiseKeys: []string{"data_ai_ml"}},
	{SkillText: "retrieval-augmented generation (rag)", DisplayText: "Retrieval-Augmented Generation (RAG)", ExpertiseKeys: []string{"data_ai_ml"}},
	{SkillText: "vector databases", DisplayText: "Vector Databases", ExpertiseKeys: []string{"data_ai_ml"}},
	{SkillText: "pinecone", DisplayText: "Pinecone", ExpertiseKeys: []string{"data_ai_ml"}},
	{SkillText: "weaviate", DisplayText: "Weaviate", ExpertiseKeys: []string{"data_ai_ml"}},
	{SkillText: "qdrant", DisplayText: "Qdrant", ExpertiseKeys: []string{"data_ai_ml"}},
	{SkillText: "mlops", DisplayText: "MLOps", ExpertiseKeys: []string{"data_ai_ml"}},
	{SkillText: "mlflow", DisplayText: "MLflow", ExpertiseKeys: []string{"data_ai_ml"}},
	{SkillText: "dvc", DisplayText: "DVC", ExpertiseKeys: []string{"data_ai_ml"}},
	{SkillText: "apache airflow", DisplayText: "Apache Airflow", ExpertiseKeys: []string{"data_ai_ml"}},
	{SkillText: "dbt", DisplayText: "dbt", ExpertiseKeys: []string{"data_ai_ml"}},
	{SkillText: "apache spark", DisplayText: "Apache Spark", ExpertiseKeys: []string{"data_ai_ml"}},
	{SkillText: "apache kafka", DisplayText: "Apache Kafka", ExpertiseKeys: []string{"data_ai_ml", "development"}},
	{SkillText: "snowflake", DisplayText: "Snowflake", ExpertiseKeys: []string{"data_ai_ml"}},
	{SkillText: "bigquery", DisplayText: "BigQuery", ExpertiseKeys: []string{"data_ai_ml"}},
	{SkillText: "databricks", DisplayText: "Databricks", ExpertiseKeys: []string{"data_ai_ml"}},
	{SkillText: "natural language processing", DisplayText: "Natural Language Processing", ExpertiseKeys: []string{"data_ai_ml"}},
	{SkillText: "computer vision", DisplayText: "Computer Vision", ExpertiseKeys: []string{"data_ai_ml"}},
	{SkillText: "opencv", DisplayText: "OpenCV", ExpertiseKeys: []string{"data_ai_ml"}},
	{SkillText: "time series forecasting", DisplayText: "Time Series Forecasting", ExpertiseKeys: []string{"data_ai_ml"}},
	{SkillText: "recommender systems", DisplayText: "Recommender Systems", ExpertiseKeys: []string{"data_ai_ml"}},
	{SkillText: "data engineering", DisplayText: "Data Engineering", ExpertiseKeys: []string{"data_ai_ml"}},
	{SkillText: "data visualization", DisplayText: "Data Visualization", ExpertiseKeys: []string{"data_ai_ml"}},
	{SkillText: "tableau", DisplayText: "Tableau", ExpertiseKeys: []string{"data_ai_ml", "marketing_growth"}},
	{SkillText: "power bi", DisplayText: "Power BI", ExpertiseKeys: []string{"data_ai_ml", "finance_accounting"}},
	{SkillText: "looker", DisplayText: "Looker", ExpertiseKeys: []string{"data_ai_ml", "marketing_growth"}},
	{SkillText: "sql", DisplayText: "SQL", ExpertiseKeys: []string{"data_ai_ml", "development"}},

	// ---------------------------------------------------------------
	// design_ui_ux (target ~30)
	// ---------------------------------------------------------------
	{SkillText: "figma", DisplayText: "Figma", ExpertiseKeys: []string{"design_ui_ux", "product_ux_research"}},
	{SkillText: "sketch", DisplayText: "Sketch", ExpertiseKeys: []string{"design_ui_ux"}},
	{SkillText: "adobe xd", DisplayText: "Adobe XD", ExpertiseKeys: []string{"design_ui_ux"}},
	{SkillText: "penpot", DisplayText: "Penpot", ExpertiseKeys: []string{"design_ui_ux"}},
	{SkillText: "invision", DisplayText: "InVision", ExpertiseKeys: []string{"design_ui_ux"}},
	{SkillText: "wireframing", DisplayText: "Wireframing", ExpertiseKeys: []string{"design_ui_ux", "product_ux_research"}},
	{SkillText: "prototyping", DisplayText: "Prototyping", ExpertiseKeys: []string{"design_ui_ux", "product_ux_research"}},
	{SkillText: "design systems", DisplayText: "Design Systems", ExpertiseKeys: []string{"design_ui_ux"}},
	{SkillText: "interaction design", DisplayText: "Interaction Design", ExpertiseKeys: []string{"design_ui_ux"}},
	{SkillText: "information architecture", DisplayText: "Information Architecture", ExpertiseKeys: []string{"design_ui_ux", "product_ux_research"}},
	{SkillText: "user flows", DisplayText: "User Flows", ExpertiseKeys: []string{"design_ui_ux", "product_ux_research"}},
	{SkillText: "mobile app design", DisplayText: "Mobile App Design", ExpertiseKeys: []string{"design_ui_ux"}},
	{SkillText: "responsive web design", DisplayText: "Responsive Web Design", ExpertiseKeys: []string{"design_ui_ux"}},
	{SkillText: "landing page design", DisplayText: "Landing Page Design", ExpertiseKeys: []string{"design_ui_ux", "marketing_growth"}},
	{SkillText: "saas product design", DisplayText: "SaaS Product Design", ExpertiseKeys: []string{"design_ui_ux"}},
	{SkillText: "dashboard design", DisplayText: "Dashboard Design", ExpertiseKeys: []string{"design_ui_ux"}},
	{SkillText: "micro-interactions", DisplayText: "Micro-interactions", ExpertiseKeys: []string{"design_ui_ux"}},
	{SkillText: "typography", DisplayText: "Typography", ExpertiseKeys: []string{"design_ui_ux"}},
	{SkillText: "color theory", DisplayText: "Color Theory", ExpertiseKeys: []string{"design_ui_ux"}},
	{SkillText: "branding", DisplayText: "Branding", ExpertiseKeys: []string{"design_ui_ux", "marketing_growth"}},
	{SkillText: "visual identity", DisplayText: "Visual Identity", ExpertiseKeys: []string{"design_ui_ux", "marketing_growth"}},
	{SkillText: "logo design", DisplayText: "Logo Design", ExpertiseKeys: []string{"design_ui_ux"}},
	{SkillText: "illustration", DisplayText: "Illustration", ExpertiseKeys: []string{"design_ui_ux"}},
	{SkillText: "adobe illustrator", DisplayText: "Adobe Illustrator", ExpertiseKeys: []string{"design_ui_ux"}},
	{SkillText: "adobe photoshop", DisplayText: "Adobe Photoshop", ExpertiseKeys: []string{"design_ui_ux", "photo_audiovisual"}},
	{SkillText: "canva", DisplayText: "Canva", ExpertiseKeys: []string{"design_ui_ux", "marketing_growth"}},
	{SkillText: "affinity designer", DisplayText: "Affinity Designer", ExpertiseKeys: []string{"design_ui_ux"}},
	{SkillText: "icon design", DisplayText: "Icon Design", ExpertiseKeys: []string{"design_ui_ux"}},
	{SkillText: "print design", DisplayText: "Print Design", ExpertiseKeys: []string{"design_ui_ux"}},
	{SkillText: "packaging design", DisplayText: "Packaging Design", ExpertiseKeys: []string{"design_ui_ux"}},
	{SkillText: "editorial design", DisplayText: "Editorial Design", ExpertiseKeys: []string{"design_ui_ux"}},
	{SkillText: "data-viz design", DisplayText: "Data-Viz Design", ExpertiseKeys: []string{"design_ui_ux", "data_ai_ml"}},
	{SkillText: "accessibility design", DisplayText: "Accessibility Design", ExpertiseKeys: []string{"design_ui_ux"}},

	// ---------------------------------------------------------------
	// design_3d_animation (target ~20)
	// ---------------------------------------------------------------
	{SkillText: "blender", DisplayText: "Blender", ExpertiseKeys: []string{"design_3d_animation"}},
	{SkillText: "cinema 4d", DisplayText: "Cinema 4D", ExpertiseKeys: []string{"design_3d_animation", "video_motion"}},
	{SkillText: "autodesk maya", DisplayText: "Autodesk Maya", ExpertiseKeys: []string{"design_3d_animation"}},
	{SkillText: "3ds max", DisplayText: "3ds Max", ExpertiseKeys: []string{"design_3d_animation"}},
	{SkillText: "zbrush", DisplayText: "ZBrush", ExpertiseKeys: []string{"design_3d_animation"}},
	{SkillText: "houdini", DisplayText: "Houdini", ExpertiseKeys: []string{"design_3d_animation", "video_motion"}},
	{SkillText: "substance painter", DisplayText: "Substance Painter", ExpertiseKeys: []string{"design_3d_animation"}},
	{SkillText: "substance designer", DisplayText: "Substance Designer", ExpertiseKeys: []string{"design_3d_animation"}},
	{SkillText: "marvelous designer", DisplayText: "Marvelous Designer", ExpertiseKeys: []string{"design_3d_animation"}},
	{SkillText: "3d modeling", DisplayText: "3D Modeling", ExpertiseKeys: []string{"design_3d_animation"}},
	{SkillText: "3d sculpting", DisplayText: "3D Sculpting", ExpertiseKeys: []string{"design_3d_animation"}},
	{SkillText: "character rigging", DisplayText: "Character Rigging", ExpertiseKeys: []string{"design_3d_animation"}},
	{SkillText: "character animation", DisplayText: "Character Animation", ExpertiseKeys: []string{"design_3d_animation"}},
	{SkillText: "3d texturing", DisplayText: "3D Texturing", ExpertiseKeys: []string{"design_3d_animation"}},
	{SkillText: "3d rendering", DisplayText: "3D Rendering", ExpertiseKeys: []string{"design_3d_animation"}},
	{SkillText: "unreal engine virtual production", DisplayText: "Unreal Engine Virtual Production", ExpertiseKeys: []string{"design_3d_animation", "video_motion"}},
	{SkillText: "arnold renderer", DisplayText: "Arnold Renderer", ExpertiseKeys: []string{"design_3d_animation"}},
	{SkillText: "redshift renderer", DisplayText: "Redshift Renderer", ExpertiseKeys: []string{"design_3d_animation"}},
	{SkillText: "octane renderer", DisplayText: "Octane Renderer", ExpertiseKeys: []string{"design_3d_animation"}},
	{SkillText: "3d printing design", DisplayText: "3D Printing Design", ExpertiseKeys: []string{"design_3d_animation"}},
	{SkillText: "architectural visualization", DisplayText: "Architectural Visualization", ExpertiseKeys: []string{"design_3d_animation"}},
	{SkillText: "product 3d visualization", DisplayText: "Product 3D Visualization", ExpertiseKeys: []string{"design_3d_animation"}},
	{SkillText: "motion capture", DisplayText: "Motion Capture", ExpertiseKeys: []string{"design_3d_animation", "video_motion"}},
	{SkillText: "2d animation", DisplayText: "2D Animation", ExpertiseKeys: []string{"design_3d_animation", "video_motion"}},
	{SkillText: "stop motion", DisplayText: "Stop Motion", ExpertiseKeys: []string{"design_3d_animation", "video_motion"}},

	// ---------------------------------------------------------------
	// video_motion (target ~20)
	// ---------------------------------------------------------------
	{SkillText: "adobe premiere pro", DisplayText: "Adobe Premiere Pro", ExpertiseKeys: []string{"video_motion"}},
	{SkillText: "final cut pro", DisplayText: "Final Cut Pro", ExpertiseKeys: []string{"video_motion"}},
	{SkillText: "davinci resolve", DisplayText: "DaVinci Resolve", ExpertiseKeys: []string{"video_motion"}},
	{SkillText: "adobe after effects", DisplayText: "Adobe After Effects", ExpertiseKeys: []string{"video_motion"}},
	{SkillText: "motion graphics", DisplayText: "Motion Graphics", ExpertiseKeys: []string{"video_motion"}},
	{SkillText: "video editing", DisplayText: "Video Editing", ExpertiseKeys: []string{"video_motion"}},
	{SkillText: "color grading", DisplayText: "Color Grading", ExpertiseKeys: []string{"video_motion"}},
	{SkillText: "visual effects (vfx)", DisplayText: "Visual Effects (VFX)", ExpertiseKeys: []string{"video_motion"}},
	{SkillText: "compositing", DisplayText: "Compositing", ExpertiseKeys: []string{"video_motion"}},
	{SkillText: "nuke", DisplayText: "Nuke", ExpertiseKeys: []string{"video_motion"}},
	{SkillText: "explainer videos", DisplayText: "Explainer Videos", ExpertiseKeys: []string{"video_motion", "marketing_growth"}},
	{SkillText: "social media video", DisplayText: "Social Media Video", ExpertiseKeys: []string{"video_motion", "marketing_growth"}},
	{SkillText: "youtube editing", DisplayText: "YouTube Editing", ExpertiseKeys: []string{"video_motion"}},
	{SkillText: "reels & tiktok editing", DisplayText: "Reels & TikTok Editing", ExpertiseKeys: []string{"video_motion", "marketing_growth"}},
	{SkillText: "video storyboarding", DisplayText: "Video Storyboarding", ExpertiseKeys: []string{"video_motion"}},
	{SkillText: "lottie animation", DisplayText: "Lottie Animation", ExpertiseKeys: []string{"video_motion", "design_ui_ux"}},
	{SkillText: "kinetic typography", DisplayText: "Kinetic Typography", ExpertiseKeys: []string{"video_motion"}},
	{SkillText: "title design", DisplayText: "Title Design", ExpertiseKeys: []string{"video_motion"}},
	{SkillText: "livestream production", DisplayText: "Livestream Production", ExpertiseKeys: []string{"video_motion"}},
	{SkillText: "obs studio", DisplayText: "OBS Studio", ExpertiseKeys: []string{"video_motion"}},
	{SkillText: "3d motion design", DisplayText: "3D Motion Design", ExpertiseKeys: []string{"video_motion", "design_3d_animation"}},
	{SkillText: "rotoscoping", DisplayText: "Rotoscoping", ExpertiseKeys: []string{"video_motion"}},

	// ---------------------------------------------------------------
	// photo_audiovisual (target ~20)
	// ---------------------------------------------------------------
	{SkillText: "photography", DisplayText: "Photography", ExpertiseKeys: []string{"photo_audiovisual"}},
	{SkillText: "portrait photography", DisplayText: "Portrait Photography", ExpertiseKeys: []string{"photo_audiovisual"}},
	{SkillText: "product photography", DisplayText: "Product Photography", ExpertiseKeys: []string{"photo_audiovisual"}},
	{SkillText: "event photography", DisplayText: "Event Photography", ExpertiseKeys: []string{"photo_audiovisual"}},
	{SkillText: "wedding photography", DisplayText: "Wedding Photography", ExpertiseKeys: []string{"photo_audiovisual"}},
	{SkillText: "fashion photography", DisplayText: "Fashion Photography", ExpertiseKeys: []string{"photo_audiovisual"}},
	{SkillText: "real estate photography", DisplayText: "Real Estate Photography", ExpertiseKeys: []string{"photo_audiovisual"}},
	{SkillText: "drone photography", DisplayText: "Drone Photography", ExpertiseKeys: []string{"photo_audiovisual"}},
	{SkillText: "photo retouching", DisplayText: "Photo Retouching", ExpertiseKeys: []string{"photo_audiovisual"}},
	{SkillText: "adobe lightroom", DisplayText: "Adobe Lightroom", ExpertiseKeys: []string{"photo_audiovisual"}},
	{SkillText: "capture one", DisplayText: "Capture One", ExpertiseKeys: []string{"photo_audiovisual"}},
	{SkillText: "studio lighting", DisplayText: "Studio Lighting", ExpertiseKeys: []string{"photo_audiovisual"}},
	{SkillText: "videography", DisplayText: "Videography", ExpertiseKeys: []string{"photo_audiovisual", "video_motion"}},
	{SkillText: "documentary filmmaking", DisplayText: "Documentary Filmmaking", ExpertiseKeys: []string{"photo_audiovisual", "video_motion"}},
	{SkillText: "sound design", DisplayText: "Sound Design", ExpertiseKeys: []string{"photo_audiovisual"}},
	{SkillText: "audio mixing", DisplayText: "Audio Mixing", ExpertiseKeys: []string{"photo_audiovisual"}},
	{SkillText: "audio mastering", DisplayText: "Audio Mastering", ExpertiseKeys: []string{"photo_audiovisual"}},
	{SkillText: "podcast production", DisplayText: "Podcast Production", ExpertiseKeys: []string{"photo_audiovisual"}},
	{SkillText: "voice-over recording", DisplayText: "Voice-over Recording", ExpertiseKeys: []string{"photo_audiovisual"}},
	{SkillText: "foley", DisplayText: "Foley", ExpertiseKeys: []string{"photo_audiovisual"}},
	{SkillText: "pro tools", DisplayText: "Pro Tools", ExpertiseKeys: []string{"photo_audiovisual"}},
	{SkillText: "logic pro", DisplayText: "Logic Pro", ExpertiseKeys: []string{"photo_audiovisual"}},
	{SkillText: "ableton live", DisplayText: "Ableton Live", ExpertiseKeys: []string{"photo_audiovisual"}},

	// ---------------------------------------------------------------
	// marketing_growth (target ~40)
	// ---------------------------------------------------------------
	{SkillText: "seo", DisplayText: "SEO", ExpertiseKeys: []string{"marketing_growth"}},
	{SkillText: "technical seo", DisplayText: "Technical SEO", ExpertiseKeys: []string{"marketing_growth"}},
	{SkillText: "sea", DisplayText: "SEA", ExpertiseKeys: []string{"marketing_growth"}},
	{SkillText: "google ads", DisplayText: "Google Ads", ExpertiseKeys: []string{"marketing_growth"}},
	{SkillText: "meta ads", DisplayText: "Meta Ads", ExpertiseKeys: []string{"marketing_growth"}},
	{SkillText: "linkedin ads", DisplayText: "LinkedIn Ads", ExpertiseKeys: []string{"marketing_growth"}},
	{SkillText: "tiktok ads", DisplayText: "TikTok Ads", ExpertiseKeys: []string{"marketing_growth"}},
	{SkillText: "youtube ads", DisplayText: "YouTube Ads", ExpertiseKeys: []string{"marketing_growth"}},
	{SkillText: "display advertising", DisplayText: "Display Advertising", ExpertiseKeys: []string{"marketing_growth"}},
	{SkillText: "programmatic advertising", DisplayText: "Programmatic Advertising", ExpertiseKeys: []string{"marketing_growth"}},
	{SkillText: "content marketing", DisplayText: "Content Marketing", ExpertiseKeys: []string{"marketing_growth"}},
	{SkillText: "content strategy", DisplayText: "Content Strategy", ExpertiseKeys: []string{"marketing_growth", "writing_translation"}},
	{SkillText: "copywriting", DisplayText: "Copywriting", ExpertiseKeys: []string{"marketing_growth", "writing_translation"}},
	{SkillText: "ugc production", DisplayText: "UGC Production", ExpertiseKeys: []string{"marketing_growth"}},
	{SkillText: "social media management", DisplayText: "Social Media Management", ExpertiseKeys: []string{"marketing_growth"}},
	{SkillText: "community management", DisplayText: "Community Management", ExpertiseKeys: []string{"marketing_growth"}},
	{SkillText: "influencer marketing", DisplayText: "Influencer Marketing", ExpertiseKeys: []string{"marketing_growth"}},
	{SkillText: "email marketing", DisplayText: "Email Marketing", ExpertiseKeys: []string{"marketing_growth"}},
	{SkillText: "marketing automation", DisplayText: "Marketing Automation", ExpertiseKeys: []string{"marketing_growth"}},
	{SkillText: "hubspot", DisplayText: "HubSpot", ExpertiseKeys: []string{"marketing_growth", "business_dev_sales"}},
	{SkillText: "brevo", DisplayText: "Brevo", ExpertiseKeys: []string{"marketing_growth"}},
	{SkillText: "mailchimp", DisplayText: "Mailchimp", ExpertiseKeys: []string{"marketing_growth"}},
	{SkillText: "klaviyo", DisplayText: "Klaviyo", ExpertiseKeys: []string{"marketing_growth"}},
	{SkillText: "customer.io", DisplayText: "Customer.io", ExpertiseKeys: []string{"marketing_growth"}},
	{SkillText: "google analytics", DisplayText: "Google Analytics", ExpertiseKeys: []string{"marketing_growth", "data_ai_ml"}},
	{SkillText: "plausible analytics", DisplayText: "Plausible Analytics", ExpertiseKeys: []string{"marketing_growth"}},
	{SkillText: "mixpanel", DisplayText: "Mixpanel", ExpertiseKeys: []string{"marketing_growth", "product_ux_research"}},
	{SkillText: "amplitude", DisplayText: "Amplitude", ExpertiseKeys: []string{"marketing_growth", "product_ux_research"}},
	{SkillText: "segment", DisplayText: "Segment", ExpertiseKeys: []string{"marketing_growth"}},
	{SkillText: "google tag manager", DisplayText: "Google Tag Manager", ExpertiseKeys: []string{"marketing_growth"}},
	{SkillText: "growth hacking", DisplayText: "Growth Hacking", ExpertiseKeys: []string{"marketing_growth"}},
	{SkillText: "a/b testing", DisplayText: "A/B Testing", ExpertiseKeys: []string{"marketing_growth", "product_ux_research"}},
	{SkillText: "conversion rate optimization", DisplayText: "Conversion Rate Optimization", ExpertiseKeys: []string{"marketing_growth"}},
	{SkillText: "inbound marketing", DisplayText: "Inbound Marketing", ExpertiseKeys: []string{"marketing_growth"}},
	{SkillText: "account-based marketing", DisplayText: "Account-Based Marketing", ExpertiseKeys: []string{"marketing_growth", "business_dev_sales"}},
	{SkillText: "brand strategy", DisplayText: "Brand Strategy", ExpertiseKeys: []string{"marketing_growth", "consulting_strategy"}},
	{SkillText: "public relations", DisplayText: "Public Relations", ExpertiseKeys: []string{"marketing_growth"}},
	{SkillText: "event marketing", DisplayText: "Event Marketing", ExpertiseKeys: []string{"marketing_growth"}},
	{SkillText: "affiliate marketing", DisplayText: "Affiliate Marketing", ExpertiseKeys: []string{"marketing_growth"}},
	{SkillText: "app store optimization", DisplayText: "App Store Optimization", ExpertiseKeys: []string{"marketing_growth"}},
	{SkillText: "landing page optimization", DisplayText: "Landing Page Optimization", ExpertiseKeys: []string{"marketing_growth"}},
	{SkillText: "semrush", DisplayText: "Semrush", ExpertiseKeys: []string{"marketing_growth"}},
	{SkillText: "ahrefs", DisplayText: "Ahrefs", ExpertiseKeys: []string{"marketing_growth"}},
}
