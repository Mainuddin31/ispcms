cd ~/developer/ispcms

# Remove the stale lock
rm -f .git/index.lock

# Stage all changes
git add -A

# Commit
git commit -m "feat: add Visiting module (visit scheduling, schedule view, dashboard integration fixed)
"

# Push
git push origin main