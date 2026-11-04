# SPDX-License-Identifier: Apache-2.0
# Squash current tree into one commit (keep latest message) and force-push to origin/master only.
# Usage (from repo root):
#   powershell -ExecutionPolicy Bypass -File .\scripts\force-push-single-commit.ps1

$ErrorActionPreference = "Stop"
Set-Location (Resolve-Path (Join-Path $PSScriptRoot ".."))

Write-Host "Remote origin:" (git remote get-url origin)
Write-Host "Current HEAD:" (git log -1 --oneline)
Write-Host "Branch:" (git rev-parse --abbrev-ref HEAD)

$msg = git log -1 --format=%B
if ([string]::IsNullOrWhiteSpace($msg)) {
    throw "empty commit message from HEAD"
}

Write-Host "Using commit message:"
Write-Host $msg

git checkout --orphan tmp-single-commit
git add -A
git commit -m "$msg"
git branch -D master
git branch -M master

Write-Host "Local history after squash:"
git log --oneline
Write-Host "Commit count:" (git rev-list --count HEAD)

Write-Host "Force pushing to origin master (not HYGON-AI)..."
git push --force origin master

Write-Host "Done."
git status -sb
git log -3 --oneline
