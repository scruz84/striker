import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMe } from '../../api/hooks'

export function HomePage() {
  const { data: loggedUser, isLoading } = useMe()
  const navigate = useNavigate()

  useEffect(() => {
    if (loggedUser) {
      navigate('/dashboard')
    }
  }, [loggedUser, navigate])

  if (isLoading) {
    return <div>Loading...</div>
  }

  return <div>Welcome to Striker Football Manager!</div>
}
