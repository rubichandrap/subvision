// The caption faces, for the editor's live preview only. The rendered video
// uses the same families bundled server-side via @fontsource (see the vfx
// templates) — these next/font instances just make the preview honest.
import { Anton, Bebas_Neue, Inter, Montserrat, Oswald, Poppins } from 'next/font/google';

import type { CaptionFont } from '@/lib/edit-spec';

// next/font loaders must each be a module-scope const.
const montserrat = Montserrat({
  subsets: ['latin'],
  weight: ['700', '800'],
  display: 'swap',
});
const inter = Inter({ subsets: ['latin'], weight: ['700', '900'], display: 'swap' });
const poppins = Poppins({ subsets: ['latin'], weight: ['600', '700'], display: 'swap' });
const oswald = Oswald({ subsets: ['latin'], weight: ['500', '700'], display: 'swap' });
const bebasNeue = Bebas_Neue({ subsets: ['latin'], weight: '400', display: 'swap' });
const anton = Anton({ subsets: ['latin'], weight: '400', display: 'swap' });

const captionFontInstances = {
  Montserrat: montserrat,
  Inter: inter,
  Poppins: poppins,
  Oswald: oswald,
  'Bebas Neue': bebasNeue,
  Anton: anton,
};

// The CSS font-family value to preview a caption font with.
export function captionFontFamily(name: CaptionFont): string {
  return `${captionFontInstances[name].style.fontFamily}, sans-serif`;
}
