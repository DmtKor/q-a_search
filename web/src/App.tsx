import { Routes, Route } from 'react-router-dom'
import { ErrorProvider } from '@/context/error'
import { SettingsProvider } from '@/store/settings'
import { ErrorBanner } from '@/components/ErrorBanner'
import { Sidebar } from '@/components/Sidebar'
import { Settings } from '@/pages/Settings'
import { Search } from '@/pages/Search'
import { CasesList, CaseDetail, CaseCreate, CaseEdit, CaseStatus, CaseImport, CaseExport } from '@/pages/Cases'
import { TicketsList, TicketDetail, TicketCreate, TicketEdit, TicketConvert } from '@/pages/Tickets'
import { AppsList, AppDetail, AppCreate } from '@/pages/Apps'

export default function App() {
  return (
    <SettingsProvider>
      <ErrorProvider>
        <div className="app-layout">
          <Sidebar />
          <main className="main">
            <ErrorBanner />
            <Routes>
              <Route path="/" element={<Search />} />
              <Route path="/search" element={<Search />} />
              <Route path="/settings" element={<Settings />} />
              <Route path="/cases" element={<CasesList />} />
              <Route path="/cases/import" element={<CaseImport />} />
              <Route path="/cases/export" element={<CaseExport />} />
              <Route path="/cases/new" element={<CaseCreate />} />
              <Route path="/cases/:id" element={<CaseDetail />} />
              <Route path="/cases/:id/edit" element={<CaseEdit />} />
              <Route path="/cases/:id/status" element={<CaseStatus />} />
              <Route path="/tickets" element={<TicketsList />} />
              <Route path="/tickets/new" element={<TicketCreate />} />
              <Route path="/tickets/:id" element={<TicketDetail />} />
              <Route path="/tickets/:id/edit" element={<TicketEdit />} />
              <Route path="/tickets/:id/convert" element={<TicketConvert />} />
              <Route path="/apps" element={<AppsList />} />
              <Route path="/apps/new" element={<AppCreate />} />
              <Route path="/apps/:id" element={<AppDetail />} />
            </Routes>
          </main>
        </div>
      </ErrorProvider>
    </SettingsProvider>
  )
}
