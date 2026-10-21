import { useEffect } from 'react'
import { useBlocker } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useConfirm } from '../../components/confirmDialogContext'

/** Warns before leaving a page with unsaved edits: the browser's own prompt on a tab close or
 * reload, and an in-app confirm dialog when navigating to another route. */
export function useUnsavedChangesGuard(dirty: boolean) {
  const { t } = useTranslation()
  const confirm = useConfirm()
  const blocker = useBlocker(dirty)

  useEffect(() => {
    if (!dirty) {
      return
    }
    const handler = (event: BeforeUnloadEvent) => {
      event.preventDefault()
    }
    window.addEventListener('beforeunload', handler)
    return () => window.removeEventListener('beforeunload', handler)
  }, [dirty])

  useEffect(() => {
    if (blocker.state !== 'blocked') {
      return
    }
    void confirm({
      title: t('lineup.leaveTitle'),
      message: t('lineup.leaveMessage'),
      confirmLabel: t('lineup.leaveConfirm'),
      destructive: true,
    }).then((leave) => {
      if (leave) {
        blocker.proceed()
      } else {
        blocker.reset()
      }
    })
    // blocker.state is the only trigger - proceed/reset are stable per blocked navigation.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [blocker.state])
}
