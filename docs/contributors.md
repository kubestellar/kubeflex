# Developing Kubeflex

## Prereqs

- go version >= go1.19.2 
- git
- make 
- gcc
- docker
- kind

Make sure that `${HOME}/go/bin` is in your `$PATH`.

## How to build kubeflex from source

Clone the repo, build the binaries and add them to your path:

```shell
git clone https://github.com/kubestellar/kubeflex.git
cd kubeflex
make build-all
export PATH=$(pwd)/bin:$PATH
```
### Steps to make release

1. delete branch "brew" from https://github.com/kubestellar/kubeflex 
2. git checkout <release branch> # e.g. release-0.2
3. Run the rebase from main
```
gitr(){
  CURRENT=$(git rev-parse --abbrev-ref HEAD)
  echo "rebasing $CURRENT"
  git checkout main && git fetch upstream && git merge upstream/main && git checkout $CURRENT && git rebase main
}
```
4. git push upstream <release branch> # e.g. release-0.2
5. check existing tags e.g.,
```
git tag 
v0.1.0
v0.1.1
v0.2.0
v0.2.1
```
6. create a new tag e.g.
```
git tag v0.2.2
```
7. Push the tag upstream
```
git push upstream --tag v0.2.2
```
Finally, make a PR from the brew branch for the brew install script.