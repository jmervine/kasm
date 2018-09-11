## KASM: Kubernetes AWS Secret Manager

`kams` is a bridge between AWS Secret Manager and Kubernetes ConfigMaps and Secrets.
It's intended to allow you commit references to AWS Secret Manager keys and have them
loaded in to Kubernetes.

Current, it's pretty dumb, it makes the following assumtions:
1. It doesn't care about the file type, it will process anything that matches
   the pattern. This might be a feature.
2. The file will have a "kind" of either a "Secret" or "ConfigMap".
    - That's not totally true. If the file is "kind: Secret" then it will base64
      encode the AWS Secret Manager value for you. Otherwise it will replace the
      key in plain text.
3. It expects the following pattern for the value: `{{ secretmanager "/key/name" "default" }}`
    - `/key/name` should be replaced with your exact secretmanager secrent name.
    - `default` is optional, but currently `kasm` will add a blank value when
      the key is not found without the `-x` flag.

## Example

Assuming `/some/username` is unset and `/some/password` is `foobar`...

#### Example YAML
```
# file: config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: example-secrets
type: Opaque
data:
  username: "{{ secretmanager "/some/username" "admin" }}"
  password: "{{ secretmanager "/some/password" "change" }}"
```

#### Example Run
```
kasm config.yaml | kubectl apply -f -
```

#### Example Resulting ConfigMap
```
# file: config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: splunk-secrets
type: Opaque
data:
  username: "admin"
  password: "foobar"
```
