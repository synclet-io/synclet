export interface Column {
  key: string
  label: string
  align?: 'left' | 'right' | 'center'
  width?: string
}

export interface Tab {
  name: string
  to?: string | object
  value?: string
}

export interface DropdownItem {
  label: string
  value?: string
  active?: boolean
  onClick?: () => void
}
