/** Whether `typed` is the text that confirms deleting the account: the club's exact name (case
 * sensitive) for a user with a club, or DELETE / BORRAR (any case) for one without. Mirrors
 * backend common.DeleteConfirmationMatches, which enforces it. */
export function isDeleteConfirmation(typed: string, teamName: string): boolean {
  const text = typed.trim()
  if (teamName) return text === teamName
  const word = text.toUpperCase()
  return word === 'DELETE' || word === 'BORRAR'
}
