import { useState, useEffect, useCallback } from 'react'
import { getAccounts, getTransfers, getAuditLog } from './api'
import AccountList from './components/AccountList'
import DepositForm from './components/DepositForm'
import TransferForm from './components/TransferForm'
import TransferList from './components/TransferList'
import AuditLog from './components/AuditLog'
import LedgerEntries from './components/LedgerEntries'
import './index.css'

export default function App() {
  const [accounts, setAccounts] = useState([])
  const [transfers, setTransfers] = useState([])
  const [auditLog, setAuditLog] = useState([])

  const refreshAccounts = useCallback(async () => {
    const data = await getAccounts()
    setAccounts(data || [])
  }, [])

  const refreshTransfers = useCallback(async () => {
    const data = await getTransfers()
    setTransfers(data || [])
  }, [])

  const refreshAuditLog = useCallback(async () => {
    const data = await getAuditLog()
    setAuditLog(data || [])
  }, [])

  const refreshAll = useCallback(async () => {
    await Promise.all([refreshAccounts(), refreshTransfers(), refreshAuditLog()])
  }, [refreshAccounts, refreshTransfers, refreshAuditLog])

  useEffect(() => {
    refreshAll()
  }, [refreshAll])

  return (
    <div className="app">
      <h1>Banking Ledger</h1>
      <AccountList accounts={accounts} onRefresh={refreshAll} />
      <DepositForm accounts={accounts} onRefresh={refreshAll} />
      <TransferForm accounts={accounts} onRefresh={refreshAll} />
      <TransferList transfers={transfers} accounts={accounts} onRefresh={refreshAll} />
      <LedgerEntries accounts={accounts} />
      <AuditLog entries={auditLog} accounts={accounts} />
    </div>
  )
}
