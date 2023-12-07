#!/bin/bash
echo "> setting up git hooks for commit-msg ..."

# Assuming that you are running the script from project root
PROJECT_ROOT=`pwd`

# git commit log template
git config --local commit.template "$PROJECT_ROOT"/bin/git-hooks/gitmessage.txt
git config --local --add commit.cleanup strip

## git commit log format hook
# create hooks if not there
if [ ! -d "$PROJECT_ROOT"/.git/hooks ]
then
    mkdir "$PROJECT_ROOT"/.git/hooks
fi
#
cp "$PROJECT_ROOT"/githooks/commit-msg ${PWD}/.git/hooks
chmod +x "$PROJECT_ROOT"/githooks/commit-msg
echo "> setting up git hooks for commit-msg completed successfully"
