# !/usr/bin/python3
import argparse
import subprocess
import json

ENVS = ["development", "production"]
REGIONS = ["us-west-1", "us-east-1", "us-west-2", "us-east-2"]

def list_secrets(env, app):
    out = subprocess.check_output(["ark", "secrets", "list", "-q", f"{env}.{app}"]).decode().split()
    print(f"Read {len(out)} secrets for {env}.{app}. Secrets {out}")
    return out


def parse_applications(file_name):
    apps = []
    with open(file_name, 'r', encoding='utf-8-sig') as f:
        apps = [line.strip() for line in f.readlines()]
        print(f"Read {len(apps)} applications.")
    return apps


def aws_ssm_add_tags(env, app, secret, region, is_deploy = False):
    param_name = f"/{env}/{app}/{secret}"
    if is_deploy:
        param_name = f"{param_name}/current-deploy"

    tags = [
        {"Key": "environment", "Value":  env},
        {"Key": "application", "Value":  app},
        {"Key": "key", "Value":  secret},
    ]
    try:
        subprocess.check_call([
            "aws", "ssm", "add-tags-to-resource",
            "--region", region, 
            "--resource-type", "Parameter", 
            "--resource-id", param_name,
            "--tags", json.dumps(tags)
        ])
    except Exception as e:
        print(f"Could not add tags to {env}.{app}.{secret} with {e}")


def main(applications_file):
    apps = parse_applications(applications_file)
    for env in ENVS:
        for region in REGIONS:
            print(f"Processing apps in {env}")
            for i, app in enumerate(apps):
                s = list_secrets(env, app)
                for j, secret in enumerate(s):
                    aws_ssm_add_tags(env, app, secret, region, True)
                print(f"Tagged all secrets in {env}.{app}")

            print(f"Done with {env} in {region}")


if __name__ == '__main__':
    my_parser = argparse.ArgumentParser()
    my_parser.add_argument(
        'file_input',
        action='store',
        nargs='?',
     )

    args = my_parser.parse_args()
    main(args.file_input)