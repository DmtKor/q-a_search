import { NavLink } from 'react-router-dom'
import { useSettings } from '@/store/settings'

export function Sidebar() {
  const { accessType } = useSettings()
  const isStaff = accessType === 'staff'

  return (
    <aside className="sidebar">
      <nav>
        <NavLink to="/search">Поиск</NavLink>
        {isStaff && <NavLink to="/cases">База знаний</NavLink>}
        {isStaff && <NavLink to="/tickets">Тикеты</NavLink>}
        {isStaff && <NavLink to="/apps">Приложения</NavLink>}
        <NavLink to="/settings">Настройки</NavLink>
      </nav>
    </aside>
  )
}
