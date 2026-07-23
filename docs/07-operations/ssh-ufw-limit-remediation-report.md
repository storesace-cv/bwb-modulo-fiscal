# Relatório — remediação SSH/UFW (fiscalmod sandbox)

**Data (UTC):** 2026-07-23T15:16Z–15:20Z
**Host:** `sandbox.fiscalmod.bwb.pt` / `194.9.62.239`
**Release inalterada:** `10141af3cdd9cda16cfef46bbe5a4f0c9e522815`
**Resultado:** **RESOLVIDO**

Este relatório **não** contém passwords, tokens, chaves privadas, DSN nem material de chave — apenas fingerprints.

## Causa definitiva do `Connection refused`

| Campo | Valor |
|---|---|
| Sintoma | `ssh: connect to host 194.9.62.239 port 22: Connection refused` |
| Causa | UFW `LIMIT` em TCP/22: `recent --seconds 30 --hitcount 6` → chain `ufw-user-limit` → `REJECT` (icmp-port-unreachable), **antes** do sshd |
| Não causa | Fail2ban (0 bans do operador no momento da remediação); MaxStartups (0 drops no journal); autenticação |

Manter UFW LIMIT após D2 foi uma **decisão inadequada** para o padrão real de administração (agentes, updater, sessões independentes). Multiplexing no cliente mitigou a tempestade do updater, mas **não** resolveu o acesso SSH geral sem mux.

### Distinção operacional

| Erro | Camada | Significado |
|---|---|---|
| `Connection refused` | Firewall/UFW (antes do sshd) | LIMIT/REJECT ou porta fechada |
| `Too many authentication failures` | sshd / cliente | Múltiplas identidades do `ssh-agent` sem `IdentitiesOnly=yes` |

Correcção macOS: `IdentitiesOnly yes` + `IdentityFile ~/.ssh/digitalocean` nos Hosts fiscalmod (ver snippet versionado).

## Comparação com my-bwb-app

| Aspecto | my-bwb-app (`deploy/ssh-config.snippet`) | fiscalmod (após remediação) |
|---|---|---|
| Chave | `~/.ssh/digitalocean` | `~/.ssh/digitalocean` (mesma fingerprint) |
| `IdentitiesOnly` | yes | yes |
| Root por chave | Hosts `main-srv-*` User root | `fiscalmod-root` + `PermitRootLogin prohibit-password` |
| UFW LIMIT SSH | N/A (padrão ALLOW operacional) | **LIMIT removido** → ALLOW 22 |

## Antes → Depois

### UFW

| | Antes | Depois |
|---|---|---|
| 22/tcp | LIMIT IN (v4+v6) | ALLOW IN (v4+v6) |
| 80/443 | ALLOW | ALLOW (inalterado) |
| 5432/8080/18080 | não abertas | não abertas |
| iptables dport 22 | `recent --set` + `recent --update --seconds 30 --hitcount 6` → `ufw-user-limit` → REJECT | `-j ACCEPT` directo |

### sshd (efectivo `sshd -T`)

| Directiva | Antes | Depois |
|---|---|---|
| PermitRootLogin | `no` | `without-password` (= `prohibit-password`) |
| PasswordAuthentication | `no` | `no` |
| KbdInteractiveAuthentication | `no` | `no` |
| PubkeyAuthentication | `yes` | `yes` |
| AuthenticationMethods | `publickey` | `publickey` |
| AllowUsers | `ubuntu`, `bwb-deploy` | `root`, `ubuntu`, `bwb-deploy` |
| MaxStartups | `10:30:100` | `10:30:100` (inalterado) |
| MaxAuthTries | `3` | `3` (inalterado) |

Drop-ins contraditórios (`PasswordAuthentication yes` / `PermitRootLogin yes` em cloud-init/custom) alinhados para `no` / `prohibit-password`. `ChallengeResponseAuthentication no` declarado no hardening (OpenSSH efectivo via `KbdInteractiveAuthentication no`).

### Fingerprints autorizadas (sem conteúdo)

| Local | Fingerprint |
|---|---|
| Operador `~/.ssh/digitalocean` | `SHA256:FpiaaXl5K3m1Vr0UYbp+aoLKJkLcTyE3LAwvTHMrDqY` |
| `/home/ubuntu/.ssh/authorized_keys` | `SHA256:FpiaaXl5K3m1Vr0UYbp+aoLKJkLcTyE3LAwvTHMrDqY` |
| `/root/.ssh/authorized_keys` | `SHA256:FpiaaXl5K3m1Vr0UYbp+aoLKJkLcTyE3LAwvTHMrDqY` |

Permissões: `.ssh` 0700, `authorized_keys` 0600, ownership correcto.

## Regra final

**ALLOW 22** + autenticação **exclusivamente por chave** + **root password proibida** + **Fail2ban activo**.
Multiplexing é apenas optimização — **não** requisito de acesso.

## Testes (sem ControlMaster / ControlPersist)

| Teste | Resultado |
|---|---|
| `ssh -i ~/.ssh/digitalocean ubuntu@194.9.62.239 true` | OK |
| `ssh -i ~/.ssh/digitalocean root@194.9.62.239 true` | OK |
| `ssh fiscalmod-sandbox true` | OK |
| `ssh fiscalmod-root true` | OK |
| 20 sequenciais ubuntu | **20/20** |
| 20 sequenciais root | **20/20** |
| 5 simultâneas ubuntu | **5/5** |
| 5 simultâneas root | **5/5** |
| Root com `PreferredAuthentications=password` (BatchMode) | `Permission denied (publickey)` — password **não** oferecida |
| Chave não autorizada (config isolada `-F`) | `Permission denied (publickey)` |
| Connection refused | **0** |
| Too many authentication failures | **0** |

## Segurança pós-alteração

| Check | Estado |
|---|---|
| UFW activo | sim · 22 ALLOW · 80/443 ALLOW |
| Fail2ban sshd | activo · Currently banned: 0 |
| sshd | active · NRestarts=0 · reload (sem restart/reboot) |
| MaxStartups drops (2h) | 0 |
| Health HTTPS | 200 · revision `10141af3…` inalterada |
| POST `/v1/documents` sem token | 401 |
| Nginx / PostgreSQL / API | active |
| Externo 5432 / 8080 / 18080 | closed/timeout |
| Deploy / release | nenhum |

## Backups (servidor)

`/var/backups/bwb-fiscal/ssh-ufw-fix-20260723T151625Z/` (root, 0700):

- `ufw-status-before.txt`, `iptables-before.rules`, `etc-ufw/`, `user.rules`, `user6.rules`
- `sshd_config`, `sshd_config.d/`, `sshd-T-before.txt`, `sshd-T-after.txt`, `sshd_config.d-after/`
- `ufw-status-after-ufw.txt`, `iptables-after-ufw.rules`

Cliente: `~/.ssh/config.bak.<UTC>` antes da actualização não destrutiva.

## Cliente macOS

Hosts adicionados sem remover entradas existentes: `fiscalmod-sandbox`, `fiscalmod-root`, `sandbox.fiscalmod.bwb.pt 194.9.62.239`.
Snippet versionado: `scripts/deploy/ssh-config.snippet`.
`~/.ssh/config` e `~/.ssh/digitalocean`: modo 0600.

## Incidentes

| ID | Severidade | Causa | Impacto | Resolução | Estado | Risco residual |
|---|---|---|---|---|---|---|
| INC-SSH-001 | Alta | UFW LIMIT 6 NEW/30s → REJECT | `Connection refused` em ligações independentes | Removido LIMIT; ALLOW 22; Fail2ban mantido | **RESOLVIDO** | Brute-force SSH mitigado por chave-only + Fail2ban (não por LIMIT) |
| INC-SSH-002 | Média | `PermitRootLogin no` + AllowUsers sem root; drop-ins contraditórios | Root por chave digitalocean impossível | `prohibit-password` + AllowUsers root ubuntu bwb-deploy; drop-ins alinhados; reload | **RESOLVIDO** | Root continua privilegiado — custódia da chave digitalocean |
| INC-S4-001 rate_429 | Alta | Fora de âmbito | — | Não tratado nesta sessão | Aberto | Ver relatório S4 ops |

## Critério de fecho

Cumprido: ubuntu e root autenticam por chave em ligações **independentes**, sem `Connection refused` e sem `Too many authentication failures`, com password SSH globalmente proibida.
