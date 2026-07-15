import { createFileRoute } from '@tanstack/react-router'

import { Main } from '@/components/layout'
import { CreativeStudio } from '@/features/creative-studio'

export const Route = createFileRoute('/_authenticated/imgen/')({
  component: CreativeStudioPage,
})

function CreativeStudioPage() {
  return (
    <Main className='p-0'>
      <CreativeStudio />
    </Main>
  )
}
