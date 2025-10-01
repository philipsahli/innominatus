import { getAllDocs } from '@/lib/docs'
import { DocsIndexClient } from '@/components/DocsIndexClient'

export default function DocsIndexPage() {
  const allDocs = getAllDocs()

  return <DocsIndexClient allDocs={allDocs} />
}
