import { getDocBySlug, getAllDocs } from '@/lib/docs';
import { notFound } from 'next/navigation';
import { DocPageClient } from '@/components/DocPageClient';

interface PageProps {
  params: Promise<{ slug: string[] }>;
}

export async function generateStaticParams() {
  const allDocs = getAllDocs();
  return allDocs.map((doc) => ({
    slug: doc.slug.split('/'),
  }));
}

export default async function DocPage({ params }: PageProps) {
  const { slug } = await params;
  const slugString = slug.join('/');
  const doc = getDocBySlug(slugString);

  if (!doc) {
    notFound();
  }

  // Get all docs for previous/next navigation
  const allDocs = getAllDocs();
  const currentIndex = allDocs.findIndex((d) => d.slug === slugString);
  const prevDoc = currentIndex > 0 ? allDocs[currentIndex - 1] : null;
  const nextDoc = currentIndex < allDocs.length - 1 ? allDocs[currentIndex + 1] : null;

  return <DocPageClient doc={doc} prevDoc={prevDoc} nextDoc={nextDoc} />;
}
