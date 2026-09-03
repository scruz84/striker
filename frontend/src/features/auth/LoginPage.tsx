import { GoogleLogin } from '@react-oauth/google'
import { useNavigate } from 'react-router-dom'
import { useLogin, useMe } from '../../api/hooks'

export function LoginPage() {
  const { data: loggedUser } = useMe()
  const login = useLogin()
  const navigate = useNavigate()

  if (loggedUser) {
    return <div>Already logged in as {loggedUser.name}</div>
  }

  return (
    <GoogleLogin
      onSuccess={(credentialResponse) => {
        if (!credentialResponse.credential) {
          return
        }
        login.mutate(credentialResponse.credential, {
          onSuccess: () => navigate('/dashboard'),
        })
      }}
      onError={() => {
        console.error('Login failed')
      }}
    />
  )
}
