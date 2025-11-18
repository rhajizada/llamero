FROM oven/bun:1.3 AS deps
WORKDIR /app

COPY ui/bun.lock ui/package.json ./
RUN bun install --frozen-lockfile

FROM deps AS build
COPY ui .
RUN bun run build

FROM oven/bun:1.3 AS runner
WORKDIR /app
ENV NODE_ENV=production

COPY --from=deps /app/node_modules ./node_modules
COPY --from=build /app/.next ./.next
COPY --from=build /app/public ./public
COPY --from=build /app/package.json ./package.json
COPY --from=build /app/next.config.ts ./next.config.ts
COPY --from=build /app/postcss.config.mjs ./postcss.config.mjs
COPY --from=build /app/tsconfig.json ./tsconfig.json
COPY --from=build /app/components.json ./components.json
COPY --from=build /app/app ./app

CMD ["bun", "run", "start"]
