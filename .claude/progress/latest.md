# Session 9
Date: 2026-03-27
Task: 0.9 — Generate age key (gitignored), create .sops.yaml, encrypt initial secrets.yaml per app in deployment/*/local/
Phase: 0
Status: IN_PROGRESS

## Plan
1. Generate an age key pair, save to `.age-key.txt` (already gitignored)
2. Create `.sops.yaml` with creation rules targeting `deployment/*/local/secrets.yaml`
3. Create plaintext secrets.yaml templates for mastermgmt, eureka, casdoor
4. Encrypt each with `sops --encrypt --in-place`
5. Verify: `sops -d deployment/mastermgmt/local/secrets.yaml > /dev/null`

## Infra state
No infrastructure yet.
