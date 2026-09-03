FROM node:24-slim

# ffmpeg + ffprobe for the render pass; the Chromium shared libraries
# Remotion's headless browser needs at runtime.
RUN corepack enable && apt-get update \
 && apt-get install -y --no-install-recommends ffmpeg ca-certificates \
      libnss3 libdbus-1-3 libatk1.0-0 libatk-bridge2.0-0 libcups2 \
      libxkbcommon0 libatspi2.0-0 libxcomposite1 libxdamage1 libxfixes3 \
      libxrandr2 libgbm1 libpango-1.0-0 libcairo2 libasound2 fonts-liberation \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY package.json pnpm-lock.yaml ./

RUN corepack prepare pnpm@9.15.9 --activate && pnpm install --frozen-lockfile

# The headless browser downloads at ensure time, not on the first render, so
# a running job never depends on registry access.
RUN pnpm exec remotion browser ensure

COPY . .

RUN pnpm build

CMD ["pnpm", "start"]
