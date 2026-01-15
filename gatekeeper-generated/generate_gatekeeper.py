import yaml

with open('../policy/data_access_control_gatekeeper.yaml') as f:
    policy = yaml.safe_load(f)['data_access_controls']

constraints = []

for data_type, config in policy.items():
    allowed_from = [access['from'] for access in config.get('allowed_access', [])]
    
    # Преобразуем domain3_token_policy → domain3-tcb
    allowed_ns = ["domain3-tcb" if "domain3" in d else d for d in allowed_from]

    constraint = {
        "apiVersion": "constraints.gatekeeper.sh/v1beta1",
        "kind": "SecretAccessControl",
        "metadata": {
            "name": f"restrict-{data_type.replace('_', '-')}-access"
        },
        "spec": {
            "match": {
                "kinds": [{"apiGroups": [""], "kinds": ["Pod"]}],
                "namespaces": ["domain1-untrusted", "domain2-medium"]  # где запрещено
            },
            "parameters": {
                "allowed_namespaces": allowed_ns
            }
        }
    }
    constraints.append(constraint)

for c in constraints:
    print("---")
    print(yaml.dump(c))