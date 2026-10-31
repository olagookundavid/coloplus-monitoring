import { useState } from 'react'

import { DashboardPageHeader } from '../../components/dashboard'

type SettingsSection = 'security' | 'approvals' | 'reporting'

interface SettingsFact {
  label: string
  description: string
}

const SECTIONS: Array<{ value: SettingsSection; label: string; facts: SettingsFact[] }> = [
  {
    value: 'security',
    label: 'Security',
    facts: [
      {
        label: 'Two-step verification',
        description:
          "Required for Coloplus administrators, operations supervisors and Airtel accounts. An account that hasn't enrolled yet holds no other permissions until it does.",
      },
      {
        label: 'Session timeout',
        description: 'Set in deployment configuration, not shown or changed here.',
      },
      {
        label: 'Account lockout',
        description:
          'A fixed number of failed sign-in attempts locks an account for a short period before it can try again.',
      },
    ],
  },
  {
    value: 'approvals',
    label: 'Approval rules',
    facts: [
      {
        label: 'Separate approver required',
        description:
          'Whoever submits a publication, correction or formula change can never approve it. Enforced by the database — this cannot be turned off.',
      },
      {
        label: 'Estimates labelled for Airtel',
        description:
          'Formula-based and estimated values are always shown as estimates, never as a meter reading. This cannot be turned off.',
      },
    ],
  },
  {
    value: 'reporting',
    label: 'Reporting',
    facts: [
      {
        label: 'Time zone',
        description:
          'Africa/Lagos (WAT) — used for every date, deadline and timestamp in this console.',
      },
    ],
  },
]

export function Settings() {
  const [section, setSection] = useState<SettingsSection>('security')
  const active = SECTIONS.find((s) => s.value === section) ?? SECTIONS[0]!

  return (
    <div className="flex flex-col gap-6">
      <DashboardPageHeader
        title="Settings"
        subtitle="How sign-in, approval and reporting rules work in this console — these are fixed system rules, not settings you can change here."
      />

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-4">
        <nav aria-label="Settings sections" className="lg:col-span-1">
          <ul className="flex flex-row gap-1 overflow-x-auto lg:flex-col lg:overflow-visible">
            {SECTIONS.map((s) => {
              const isActive = s.value === section
              return (
                <li key={s.value} className="shrink-0 lg:shrink">
                  <button
                    type="button"
                    onClick={() => setSection(s.value)}
                    aria-current={isActive ? 'page' : undefined}
                    className={`w-full rounded-lg px-3 py-2.5 text-left text-sm font-medium transition-colors ${
                      isActive
                        ? 'bg-primary-soft text-primary-700'
                        : 'text-text-muted hover:bg-surface-subtle hover:text-text'
                    }`}
                  >
                    {s.label}
                  </button>
                </li>
              )
            })}
          </ul>
        </nav>

        <div className="lg:col-span-3">
          <div className="rounded-lg border border-border bg-surface p-5 sm:p-6">
            <h2 className="text-lg font-bold text-text">{active.label}</h2>
            <dl className="mt-4 divide-y divide-border">
              {active.facts.map((fact) => (
                <div key={fact.label} className="py-4 first:pt-0 last:pb-0">
                  <dt className="font-medium text-text">{fact.label}</dt>
                  <dd className="mt-1 text-sm text-text-muted">{fact.description}</dd>
                </div>
              ))}
            </dl>
          </div>
        </div>
      </div>
    </div>
  )
}
