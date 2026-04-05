import { useState } from "react"
import { PageHeader } from "@/shared/components/layouts/page-header"
import { Tabs } from "@/shared/components/ui/tabs"
import { JobsTab } from "./jobs-tab"
import { ApplicationsTab } from "./applications-tab"

export function JobsPage() {
  const [activeTab, setActiveTab] = useState("jobs")

  const tabs = [
    { value: "jobs", label: "Offres" },
    { value: "applications", label: "Candidatures" },
  ]

  return (
    <div className="space-y-6">
      <PageHeader
        title="Offres d'emploi"
        description="Gestion des offres et candidatures de la marketplace"
      />

      <Tabs tabs={tabs} value={activeTab} onChange={setActiveTab} />

      {activeTab === "jobs" && <JobsTab />}
      {activeTab === "applications" && <ApplicationsTab />}
    </div>
  )
}
