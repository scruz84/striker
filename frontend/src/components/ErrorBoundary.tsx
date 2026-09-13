import { Component, type ErrorInfo, type ReactNode } from 'react'
import Button from '@mui/material/Button'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'

interface Props {
  children: ReactNode
}

interface State {
  hasError: boolean
}

// Wraps routed page content so an uncaught render error (a null/undefined field the page didn't
// expect, for instance) shows a recoverable message instead of unmounting the whole app - without
// this, React's default behavior on an uncaught render error is to tear down the entire tree,
// leaving a blank page with no way back short of a manual reload. Placed around <Outlet/> in
// Layout.tsx rather than the whole app, so the header/nav bar stay up and usable even if one
// page's content crashes.
export class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false }

  static getDerivedStateFromError(): State {
    return { hasError: true }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    // eslint-disable-next-line no-console
    console.error('Unhandled error rendering page content:', error, info.componentStack)
  }

  render() {
    if (this.state.hasError) {
      return <ErrorFallback onReset={() => this.setState({ hasError: false })} />
    }
    return this.props.children
  }
}

function ErrorFallback({ onReset }: { onReset: () => void }) {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const goHome = () => {
    onReset()
    void navigate('/')
  }

  return (
    <Stack spacing={2} sx={{ py: 6, alignItems: 'center', textAlign: 'center' }}>
      <Typography variant="h6">{t('common.somethingWentWrong')}</Typography>
      <Typography variant="body2" color="text.secondary">
        {t('common.somethingWentWrongDetail')}
      </Typography>
      <Stack direction="row" spacing={1}>
        <Button variant="outlined" onClick={onReset}>
          {t('common.tryAgain')}
        </Button>
        <Button variant="contained" onClick={goHome}>
          {t('login.backToHome')}
        </Button>
      </Stack>
    </Stack>
  )
}
