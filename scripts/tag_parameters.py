# !/usr/bin/python3
import argparse
import json
import subprocess
import sys

ENVS = ["development", "production"]
REGIONS = ["us-west-1", "us-east-1", "us-west-2", "us-east-2"]

def list_secrets(env, app):
    try:
        out = subprocess.check_output(["ark", "secrets", "list", "-q", f"{env}.{app}"], stderr=subprocess.DEVNULL).decode().split()
        print(f"Read {len(out)} secrets for {env}.{app}. Secrets {out}")
        return out
    except Exception as e:
        print(f"Error occured while listing secrets for {app} via `ark`. Make sure you are connected to VPN and your `ark` is working. SKIPPING")
        return []

def parse_applications(file_name):
    apps = []
    with open(file_name, 'r', encoding='utf-8-sig') as f:
        apps = [line.strip() for line in f.readlines()]
        print(f"Read {len(apps)} applications.")
    return apps


def aws_ssm_add_tags(env, app, secret, region, is_deploy):
    param_name = f"/{env}/{app}/{secret}"
    if is_deploy:
        param_name = f"{param_name}/current-deploy"

    tags = [
        {"Key": "environment", "Value": env},
        {"Key": "application", "Value": app},
        {"Key": "key", "Value": secret},
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

def check_aws_authentication_and_exit():
    try:
        output = subprocess.check_output(["aws", "sts", "get-caller-identity"], stderr=subprocess.DEVNULL).decode().strip()
        id = json.loads(output)
        arn = id["Arn"]
        print(f"Logged in as {arn}.")
    except Exception as e:
        print(f"Could not authenticate to AWS. Make sure to login with `saml2aws`.")
        sys.exit(0)

def main(applications_file, is_deploy):
    check_aws_authentication_and_exit() 
    apps = parse_applications(applications_file)
    
    # the deployment parameters is only deployed to us-west-1
    regions = REGIONS
    if is_deploy:
        regions = ["us-west-1"]
     
    for env in ENVS:
        for region in regions:
            print(f"Processing apps in {env}")
            for i, app in enumerate(apps):
                s = list_secrets(env, app)
                for j, secret in enumerate(s):
                    aws_ssm_add_tags(env, app, secret, region, is_deploy)
                print(f"Tagged all secrets in {env}.{app}")

            print(f"Done with {env} in {region}")


if __name__ == '__main__':
    my_parser = argparse.ArgumentParser()
    my_parser.add_argument(
        'file_input',
        action='store',
        nargs='?',
        help='A text file with names of applications in individual lines', 
     )
    
    my_parser.add_argument(
        '--deploy-params',
        '-d',
        action='store_true',
        dest='is_deploy',
        help='Boolean to indicate whether to tag deployment specific `current-deploy` parameters',
    )
    my_parser.set_defaults(is_deploy=False)

    args = my_parser.parse_args()
    main(args.file_input, args.is_deploy)