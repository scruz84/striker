import Box from '@mui/material/Box'
import LinearProgress from '@mui/material/LinearProgress'
import Typography from '@mui/material/Typography'

interface SkillBarProps {
  label: string
  value: number
}

export function SkillBar({ label, value }: SkillBarProps) {
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 0.5 }}>
      <Typography variant="body2" sx={{ width: 100, flexShrink: 0 }}>
        {label}
      </Typography>
      <Box sx={{ flexGrow: 1 }}>
        <LinearProgress variant="determinate" value={Math.min(value, 100)} />
      </Box>
      <Typography variant="body2" color="text.secondary" sx={{ width: 28, textAlign: 'right', flexShrink: 0 }}>
        {value}
      </Typography>
    </Box>
  )
}
