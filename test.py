import yaml
with open("config.yml", 'r') as stream:
    try:
        domains = yaml.safe_load(stream)
        for domain in domains['domains']:
            for domain_name in domain:
                print(domain_name)
                print(domain[domain_name]['delay'])
    except yaml.YAMLError as exc:
        print(exc)
