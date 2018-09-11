## KASM: Kubernetes AWS Secret Manager

`kams` is a bridge between AWS Secret Manager and Kubernetes ConfigMaps and Secrets.
It's intended to allow you commit references to AWS Secret Manager keys and have them
loaded in to Kubernetes.

Current, it's pretty dumb, it makes the following assumtions:
1. The file will be YAML, but it does no validation. That's okay, because k8s
   will bark at you if it's invalid.
2. The file will have a "kind" of either a "Secret" or "ConfigMap".
    - That's not totally true. If the file is "kind: Secret" then it will base64
      encode the AWS Secret Manager value for you. Otherwise it will replace the
      key in plain text.
3. It expects the following pattern for the value: `"secretmanager:{{key}}|{{default}}"`
    - Yes, the surrounding quotes are required.
    - `{{key}}` should be replaced with your exact secretmanager secrent name.
    - `{{default}}` is optional, but currently `kasm` will add a blank value when
      the key is not found.

## Example

Assuming `/some/username` is `user` and `/some/password` is `foobar`...

#### Example YAML
```
# file: config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: splunk-secrets
type: Opaque
data:
  username: "secretmanager:/some/username|admin"
  password: "secretmanager:/some/password|changeit"
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
  username: "user"
  password: "foobar"
```
