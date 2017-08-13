# Adding a new user to git-crypt

## Import Key

`gpg --import <key.asc>`

## Trust Key

Determine the name of the key you want to trust
`gpg --list-keys`

Edit the key
`gpg --edit-key email@loodse.com`

When prompted for a command, enter `trust`
`Command> trust`

When prompted for a level of trust, enter `5` for _Ultimate_ trust
`Your decision? 5`

Quit gpg
`Command> quit`

## Add key to git-crypt
Move to your _kubermatic/config_ directory
`cd config/`

Stash any uncommited changes
`git stash`

Checkout new branch
`git checkout -b git-crypt/add-user-<username>`

Add user to git-crypt
`git-crypt add-gpg-user <email@loodse.com>`

Push the new commit to origin
`git push`

Merge the branch in to develop on GitHub

## Rebase all other branches off of develop
Set up:
git checkout develop
git fetch -apt
git rebase

For each branch do the following:
git checkout <branch>
git merge develop
git push

## On the users machine

Clone the repository
`git clone git@github.com:kubermatic/config.git`

Enter the repo
`cd config`

Unlock the repository
`git-crypt unlock`
