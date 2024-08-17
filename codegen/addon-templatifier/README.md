# Addon Templatifier

## Intention

The Addon Templatifier takes an addon YAML manifest (technically any kind of
Kubernetes YAML manifest) and will transform all container images into a Go
template syntax (i.e. turn `image: "foo/bar"` into `image: {{ Image "foo/bar" }}`).

## Usage

Pipe a valid YAML file into it and the tool will replace all images with
`{{ Image ... }}`. To make the tags dynamic, use the `--dynamic` flag and specify
the repository and the desired variable name:

```bash
$ cat myaddon.yaml | ./addon-templatifier --dynamic "docker.io/alpine:myversion"
```

The above would replace all `docker.io/alpine:...` images (i.e. the tag doesn't
matter) with `{{ Image (print "docker.io/alpine:" $myversion) }}`.
