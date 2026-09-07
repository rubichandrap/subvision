# Spec: Neobrutalist Chrome UI (light + dark)

**GitHub issue:** #37

**Status:** closed 2026-09-07 — shipped (GitHub: #37 closed).

Scope: chrome UI client saja. Subtitle Style (output render, bagian Edit Spec)
tidak disentuh.

## Keputusan desain

- Brutal penuh: border 2px solid, shadow solid offset 2/4/6px tanpa blur,
  radius 0 untuk button/badge/input/tabs/select/slider/switch/toast,
  `rounded-lg` max untuk card/dropzone/dialog.
- Interaksi: hover geser -2px + shadow membesar, active tekan +2px +
  shadow susut, focus outline 3px, disabled tanpa shadow + opacity 50%.
- Palet light: paper `#FFF9EC`, tinta `#141414`, kartu `#FFFFFF`,
  muted `#F2E8CF`.
- Palet dark: espresso hangat `#2B241C` (bukan near-black), kartu `#3A3128`,
  muted `#4A4034`, teks `#FFF6E5`, border paper. Hangat, bukan `#000`.
- Aksen identik dua mode: primer kuning `#FFC900`, pink `#FF90E8`,
  biru `#90E8FF`, sukses hijau `#7DF29A`, destruktif merah `#FF5C5C`.
- Tipo: Space Grotesk 700-800 heading, Inter body, Space Mono
  badge/label/angka. Uppercase hanya eyebrow + label kecil.
- Preview video tetap netral hitam, tidak kena brutal.
- Chip count tab aktif: `bg-background text-foreground` supaya angka
  terbaca di atas tab kuning pada dark mode.

## Cakupan file

Token `client/app/globals.css` (+ font Space Mono di layout), primitif
`components/ui/*` (button, card, badge, input, textarea, tabs, select,
slider, switch, dialog, alert-dialog, sheet, progress, toast),
`app-shell`, `theme-toggle`, home (hero, dropzone, pipeline cards),
editor (stage, trim, tabs, frame/animation/style pickers), gallery
(filter tabs, kartu, state loading/empty/error), process detail
(stepper, state).

## Verifikasi

- `pnpm --dir client build` hijau, `tsc --noEmit` bersih.
- Screenshot Edge headless light + dark home terverifikasi.
- Badge count tab aktif diverifikasi via mock DOM: versi fix terbaca,
  versi lama tenggelam.
- ADR-0009 dicatat.

## Catatan proses

Pekerjaan ini dieksekusi dulu tanpa spec/issue (pelanggaran disiplin).
Spec ini ditulis retroaktif sebelum commit, sesuai teguran user 2026-09-07.
