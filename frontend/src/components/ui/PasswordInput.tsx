import { useState } from 'react'
import { EyeIcon, EyeSlashIcon } from '@heroicons/react/24/outline'

import { IconButton } from './IconButton'
import { Input, type InputProps } from './Input'

export function PasswordInput({ className = '', ...rest }: InputProps) {
  const [visible, setVisible] = useState(false)

  return (
    <div className="relative">
      <Input type={visible ? 'text' : 'password'} className={`pr-12 ${className}`} {...rest} />
      <div className="absolute inset-y-0 right-0 flex items-center">
        <IconButton
          icon={visible ? EyeSlashIcon : EyeIcon}
          label={visible ? 'Hide password' : 'Show password'}
          variant="ghost"
          onClick={() => setVisible((current) => !current)}
        />
      </div>
    </div>
  )
}
