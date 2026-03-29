"use client"

import { useState } from "react"
import { Users, Plus, Trash2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { PaymentInfoFormData } from "../types"
import type { BusinessPersonData } from "../types"

type Props = {
  data: PaymentInfoFormData
  onChange: (field: keyof PaymentInfoFormData, value: unknown) => void
}

export function BusinessPersonsSection({ data, onChange }: Props) {
  const t = useTranslations("paymentInfo")

  function addPerson(role: string) {
    const newPerson: BusinessPersonData = {
      role, firstName: "", lastName: "", dateOfBirth: "", email: "", phone: "", address: "", city: "", postalCode: "", title: "",
    }
    onChange("businessPersons", [...data.businessPersons, newPerson])
  }

  function removePerson(index: number) {
    onChange("businessPersons", data.businessPersons.filter((_, i) => i !== index))
  }

  function updatePerson(index: number, field: keyof BusinessPersonData, value: string) {
    const updated = data.businessPersons.map((p, i) => i === index ? { ...p, [field]: value } : p)
    onChange("businessPersons", updated)
  }

  const directors = data.businessPersons.filter((p) => p.role === "director")
  const owners = data.businessPersons.filter((p) => p.role === "owner")
  const executives = data.businessPersons.filter((p) => p.role === "executive")

  return (
    <div className="rounded-2xl border border-slate-100 bg-white shadow-sm dark:border-slate-700 dark:bg-slate-800/80 overflow-hidden">
      <div className="h-1 bg-gradient-to-r from-purple-500 to-indigo-500" />
      <div className="p-6 space-y-5">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-purple-100 dark:bg-purple-500/20">
            <Users className="h-5 w-5 text-purple-600 dark:text-purple-400" strokeWidth={1.5} />
          </div>
          <div>
            <h2 className="text-base font-semibold text-slate-900 dark:text-white">{t("keyPersons")}</h2>
            <p className="text-xs text-slate-500 dark:text-slate-400">{t("keyPersonsDesc")}</p>
          </div>
        </div>

        {/* 1. Représentant légal */}
        <CheckboxBlock
          checked={data.isSelfRepresentative}
          onChange={(v) => onChange("isSelfRepresentative", v)}
          label={t("iAmRepresentative")}
        />

        {/* 2. Dirigeants */}
        <CheckboxBlock
          checked={data.isSelfDirector}
          onChange={(v) => onChange("isSelfDirector", v)}
          label={t("representativeIsSoleDirector")}
        />
        {!data.isSelfDirector && (
          <PersonList
            persons={directors}
            role="director"
            label={t("directors")}
            allPersons={data.businessPersons}
            onAdd={() => addPerson("director")}
            onRemove={removePerson}
            onUpdate={updatePerson}
          />
        )}

        {/* 3. Actionnaires >25% */}
        <CheckboxBlock
          checked={data.noMajorOwners}
          onChange={(v) => onChange("noMajorOwners", v)}
          label={t("noMajorOwners")}
        />
        {!data.noMajorOwners && (
          <PersonList
            persons={owners}
            role="owner"
            label={t("owners")}
            allPersons={data.businessPersons}
            onAdd={() => addPerson("owner")}
            onRemove={removePerson}
            onUpdate={updatePerson}
          />
        )}

        {/* 4. Cadres dirigeants */}
        <CheckboxBlock
          checked={data.isSelfExecutive}
          onChange={(v) => onChange("isSelfExecutive", v)}
          label={t("representativeIsSoleExecutive")}
        />
        {!data.isSelfExecutive && (
          <PersonList
            persons={executives}
            role="executive"
            label={t("executives")}
            allPersons={data.businessPersons}
            onAdd={() => addPerson("executive")}
            onRemove={removePerson}
            onUpdate={updatePerson}
          />
        )}
      </div>
    </div>
  )
}

function CheckboxBlock({ checked, onChange, label }: { checked: boolean; onChange: (v: boolean) => void; label: string }) {
  return (
    <label className="flex items-center gap-3 cursor-pointer">
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        className="h-4 w-4 rounded border-slate-300 text-rose-500 focus:ring-rose-500"
      />
      <span className="text-sm text-slate-700 dark:text-slate-300">{label}</span>
    </label>
  )
}

function PersonList({ persons, role, label, allPersons, onAdd, onRemove, onUpdate }: {
  persons: BusinessPersonData[]
  role: string
  label: string
  allPersons: BusinessPersonData[]
  onAdd: () => void
  onRemove: (index: number) => void
  onUpdate: (index: number, field: keyof BusinessPersonData, value: string) => void
}) {
  const t = useTranslations("paymentInfo")

  return (
    <div className="ml-7 space-y-3">
      <p className="text-sm font-medium text-slate-600 dark:text-slate-400">{label}</p>
      {persons.map((person) => {
        const globalIndex = allPersons.indexOf(person)
        return (
          <div key={globalIndex} className="rounded-lg border border-slate-200 dark:border-slate-600 p-3 space-y-2">
            <div className="flex justify-between items-center">
              <span className="text-xs font-medium text-slate-500">{label} #{persons.indexOf(person) + 1}</span>
              <button type="button" onClick={() => onRemove(globalIndex)} className="text-red-500 hover:text-red-600">
                <Trash2 className="h-4 w-4" />
              </button>
            </div>
            <div className="grid grid-cols-2 gap-2">
              <MiniInput label={t("firstName")} value={person.firstName} onChange={(v) => onUpdate(globalIndex, "firstName", v)} />
              <MiniInput label={t("lastName")} value={person.lastName} onChange={(v) => onUpdate(globalIndex, "lastName", v)} />
              <MiniInput label={t("dateOfBirth")} value={person.dateOfBirth} onChange={(v) => onUpdate(globalIndex, "dateOfBirth", v)} type="date" />
              <MiniInput label="Email" value={person.email} onChange={(v) => onUpdate(globalIndex, "email", v)} type="email" />
            </div>
          </div>
        )
      })}
      <button
        type="button"
        onClick={onAdd}
        className="flex items-center gap-1.5 text-sm font-medium text-rose-600 hover:text-rose-700 dark:text-rose-400"
      >
        <Plus className="h-4 w-4" />
        {t("addPerson")}
      </button>
    </div>
  )
}

function MiniInput({ label, value, onChange, type = "text" }: {
  label: string; value: string; onChange: (v: string) => void; type?: string
}) {
  return (
    <div>
      <label className="text-xs text-slate-500 dark:text-slate-400">{label}</label>
      <input
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className={cn(
          "h-8 w-full rounded border border-slate-200 bg-white px-2 text-sm",
          "focus:border-rose-500 focus:ring-2 focus:ring-rose-500/10 focus:outline-none",
          "dark:border-slate-600 dark:bg-slate-800 dark:text-white",
        )}
      />
    </div>
  )
}
