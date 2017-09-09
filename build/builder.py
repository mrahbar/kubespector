#!/usr/bin/python2.7 -u

import argparse
import logging
import os
import re
import shutil
import subprocess
import sys
from datetime import datetime

# Packaging variables
PACKAGE_NAME = "kubespector"
PACKAGE_LICENSE = "Apache License, Version 2.0"
PACKAGE_URL = "https://github.com/mrahbar/kubernetes-inspector"
DESCRIPTION = "Management tool for Kubernetes"

prereqs = [ 'git', 'go' ]
go_vet_command = "go tool vet -composites=true ./"

targets = {
    'kubespector' : './main.go'
}

supported_builds = {
    'darwin': [ "amd64" ],
    'linux': [ "amd64", "static_amd64" ],
    'windows': [ "amd64" ]
}

supported_packages = {
    "darwin": [ "tar" ],
    "linux": [ "tar" ],
    'windows': [ "zip" ]
}

def print_banner():
    logging.info(r"""
 _  ___  _ __ ___  __  ___ ___ ________ __  ___ 
| |/ / || |  \ __/' _/| _,\ __/ _/_   _/__\| _ \
|   <| \/ | -< _|`._`.| v_/ _| \__ | || \/ | v /
|_|\_\\__/|__/___|___/|_| |___\__/ |_| \__/|_|_\

  Build Script
""")


def run(command, envs={}, allow_failure=False, shell=False):
    """Run shell command (convenience wrapper around subprocess).
    """
    if envs:
        logging.debug("'{}' with env {}".format(command, envs))
    else:
        logging.debug("'{}'".format(command))

    out = None
    system_envs = os.environ.copy()
    for k, v in envs.items():
        system_envs[k] = v

    try:
        if shell:
            out = subprocess.check_output(command, stderr=subprocess.STDOUT, shell=shell, env=system_envs)
        else:
            out = subprocess.check_output(command.split(), stderr=subprocess.STDOUT, env=system_envs)
        out = out.decode('utf-8').strip()
        # logging.debug("Command output: {}".format(out))
    except subprocess.CalledProcessError as e:
        if allow_failure:
            logging.warn("Command '{}' failed with error: {}".format(command, e.output))
            return None
        else:
            logging.error("Command '{}' failed with error: {}".format(command, e.output))
            sys.exit(1)
    except OSError as e:
        if allow_failure:
            logging.warn("Command '{}' failed with error: {}".format(command, e))
            return out
        else:
            logging.error("Command '{}' failed with error: {}".format(command, e))
            sys.exit(1)
    else:
        return out


def go_get(no_uncommitted=False):
    """Retrieve build dependencies or restore pinned dependencies.
    """
    if local_changes() and no_uncommitted:
        logging.error("There are uncommitted changes in the current directory.")
        return False
    if not check_path_for("dep"):
        logging.info("Downloading `dep`...")
        get_command = "go get -u github.com/golang/dep/cmd/dep"
        run(get_command)
    logging.info("Retrieving dependencies with `dep`...")
    sys.stdout.flush()
    run("dep ensure -v")
    return True


def increment_minor_version(version):
    """Return the version with the minor version incremented and patch
    version set to zero.
    """
    ver_list = version.split('.')
    if len(ver_list) != 3:
        logging.warn("Could not determine how to increment version '{}', will just use provided version.".format(version))
        return version
    ver_list[1] = str(int(ver_list[1]) + 1)
    ver_list[2] = str(0)
    inc_version = '.'.join(ver_list)
    logging.debug("Incremented version from '{}' to '{}'.".format(version, inc_version))
    return inc_version


def get_current_version_tag():
    """Retrieve the raw git version tag.
    """
    version = run("git describe --always --tags --abbrev=0")
    return version


def get_current_version():
    """Parse version information from git tag output.
    """
    version_tag = get_current_commit(short=True)
    return version_tag


def get_current_commit(short=False):
    """Retrieve the current git commit.
    """
    command = None
    if short:
        command = "git log --pretty=format:%h -n 1"
    else:
        command = "git rev-parse HEAD"
    out = run(command)
    return out.strip('\'\n\r ')


def get_current_branch():
    """Retrieve the current git branch.
    """
    command = "git rev-parse --abbrev-ref HEAD"
    out = run(command)
    return out.strip()


def local_changes():
    """Return True if there are local un-committed changes.
    """
    output = run("git diff-files --ignore-submodules --").strip()
    if len(output) > 0:
        return True
    return False


def get_system_arch():
    """Retrieve current system architecture.
    """
    try:
        arch = os.uname()[4]
        if arch == "x86_64":
            arch = "amd64"
        elif arch == "386":
            arch = "i386"
        elif 'arm' in arch:
            # Prevent uname from reporting full ARM arch (eg 'armv7l')
            arch = "arm"
        return arch
    except:
        if os.environ['PROGRAMFILES(X86)']:
            return "amd64"
        else:
            return "386"


def get_system_platform():
    """Retrieve current system platform.
    """
    if sys.platform.startswith("linux"):
        return "linux"
    elif sys.platform.startswith("win32"):
        return "windows"
    else:
        return sys.platform


def get_go_version():
    """Retrieve version information for Go.
    """
    out = run("go version")
    matches = re.search('go version go(\S+)', out)
    if matches is not None:
        return matches.groups()[0].strip()
    return None


def check_path_for(b):
    """Check the the user's path for the provided binary.
    """
    def is_exe(fpath):
        return os.path.isfile(fpath) and os.access(fpath, os.X_OK)

    def ext_candidates(fpath):
        yield fpath
        for ext in os.environ.get("PATHEXT", "").split(os.pathsep):
            yield fpath + ext

    fpath, fname = os.path.split(b)
    if fpath:
        if is_exe(b):
            return b
    else:
        for path in os.environ["PATH"].split(os.pathsep):
            path = path.strip('"')
            exe_file = os.path.join(path, b)
            for candidate in ext_candidates(exe_file):
                if is_exe(candidate):
                    return candidate


def check_environ(build_dir = None):
    """Check environment for common Go variables.
    """
    logging.info("Checking environment...")
    for v in [ "GOPATH", "GOBIN", "GOROOT" ]:
        logging.debug("Using '{}' for {}".format(os.environ.get(v), v))

    cwd = os.getcwd()
    if build_dir is None and os.environ.get("GOPATH") and os.environ.get("GOPATH") not in cwd:
        logging.warn("Your current directory is not under your GOPATH. This may lead to build failures.")
    return True


def check_prereqs():
    """Check user path for required dependencies.
    """
    logging.info("Checking for dependencies...")
    for req in prereqs:
        path_for = check_path_for(req)
        logging.debug("Path of {} {}...".format(req, path_for))
        if not path_for:
            logging.error("Could not find dependency: {}".format(req))
            return False
    return True


def go_list(vendor=False, relative=False):
    """
    Return a list of packages
    If vendor is False vendor package are not included
    If relative is True the package prefix defined by PACKAGE_URL is stripped
    """
    p = subprocess.Popen(["go", "list", "./..."], stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    out, err = p.communicate()
    packages = out.split('\n')
    if packages[-1] == '':
        packages = packages[:-1]
    if not vendor:
        non_vendor = []
        for p in packages:
            if '/vendor/' not in p:
                non_vendor.append(p)
        packages = non_vendor
    if relative:
        relative_pkgs = []
        for p in packages:
            r = p.replace(PACKAGE_URL, '.')
            if r != '.':
                relative_pkgs.append(r)
        packages = relative_pkgs
    return packages


def build(version=None, platform=None, arch=None, clean=False, outdir=".", tags=[], static=False):
    """
    Build each target for the specified architecture and platform.
    """
    logging.info("Starting build for {}/{}...".format(platform, arch))
    logging.info("Using Go version: {}".format(get_go_version()))
    logging.info("Using git branch: {}".format(get_current_branch()))
    logging.info("Using git commit: {}".format(get_current_commit()))
    if static:
        logging.info("Using statically-compiled output.")
    if len(tags) > 0:
        logging.info("Using build tags: {}".format(','.join(tags)))

    logging.info("Sending build output to: {}".format(outdir))
    if not os.path.exists(outdir):
        os.makedirs(outdir)
    elif clean and outdir != '/' and outdir != ".":
        logging.info("Cleaning build directory '{}' before building.".format(outdir))
        shutil.rmtree(outdir)
        os.makedirs(outdir)

    logging.info("Using version '{}' for build.".format(version))

    for target, path in targets.items():
        logging.info("Building target: {}".format(target))
        build_command = ""
        build_envs = {}

        # Handle static binary output
        if static is True or "static_" in arch:
            if "static_" in arch:
                static = True
                arch = arch.replace("static_", "")
                build_envs["CGO_ENABLED"] = "0"

        # Handle variations in architecture output
        if arch == "i386" or arch == "i686":
            arch = "386"
        elif "arm" in arch:
            arch = "arm"
        build_envs["GOOS"] = platform
        build_envs["GOARCH"] = arch

        if "arm" in arch:
            if arch == "armel":
                build_envs["GOARM"] = "5"
            elif arch == "armhf" or arch == "arm":
                build_envs["GOARM"] = "6"
            elif arch == "arm64":
                # TODO(rossmcdonald) - Verify this is the correct setting for arm64
                build_envs["GOARM"] = "7"
            else:
                logging.error("Invalid ARM architecture specified: {}".format(arch))
                logging.error("Please specify either 'armel', 'armhf', or 'arm64'.")
                return False

        target = "{}-{}".format(target,version)
        if platform == 'windows':
            target = target + '.exe'
        build_command += "go build -o {} ".format(os.path.join(outdir, target))
        if len(tags) > 0:
            build_command += "-tags {} ".format(','.join(tags))

        ldflags = "-ldflags=\"-X main.version={} -X 'main.buildDate={}' -X main.branch={} -X main.commit={}\" "
        build_command += ldflags.format(version, datetime.utcnow().strftime("%Y-%m-%d %H:%M:%S"), get_current_branch(), get_current_commit(True))

        if static:
            build_command += "-a -installsuffix cgo "
        build_command += path
        start_time = datetime.utcnow()
        run(build_command, envs=build_envs, shell=True)
        end_time = datetime.utcnow()
        logging.info("Time taken: {}s".format((end_time - start_time).total_seconds()))
    return True


def main(args):
    global PACKAGE_NAME

    if args.release and args.nightly:
        logging.error("Cannot be both a nightly and a release.")
        return 1

    if args.nightly:
        args.version = increment_minor_version(args.version)
        args.version = "{}~n{}".format(args.version,
                                       datetime.utcnow().strftime("%Y%m%d%H%M"))
    # Pre-build checks
    check_environ()
    if not check_prereqs():
        return 1
    if args.build_tags is None:
        args.build_tags = []
    else:
        args.build_tags = args.build_tags.split(',')

    orig_commit = get_current_commit(short=True)
    orig_branch = get_current_branch()

    if args.platform not in supported_builds and args.platform != 'all':
        logging.error("Invalid build platform: {}".format(args.platform))
        return 1

    build_output = {}

    if args.branch != orig_branch and args.commit != orig_commit:
        logging.error("Can only specify one branch or commit to build from.")
        return 1
    elif args.branch != orig_branch:
        logging.info("Moving to git branch: {}".format(args.branch))
        run("git checkout {}".format(args.branch))
    elif args.commit != orig_commit:
        logging.info("Moving to git commit: {}".format(args.commit))
        run("git checkout {}".format(args.commit))

    if not args.skip_dep:
        if not go_get(args.no_uncommitted):
            return 1
    else:
        logging.info("Skipping retrieval of build dependencies")

    single_build = True
    if args.platform == 'all':
        platforms = supported_builds.keys()
        single_build = False
    else:
        platforms = [args.platform]

    for platform in platforms:
        build_output.update( { platform : {} } )
        if args.arch == "all":
            single_build = False
            archs = supported_builds.get(platform)
        else:
            archs = [args.arch]

        for arch in archs:
            od = args.outdir
            if not single_build:
                od = os.path.join(args.outdir, platform, arch)
            if not build(version=args.version, platform=platform, arch=arch,
                         clean=args.clean, outdir=od, tags=args.build_tags, static=args.static):
                return 1
            build_output.get(platform).update( { arch : od } )

    if orig_branch != get_current_branch():
        logging.info("Moving back to original git branch: {}".format(orig_branch))
        run("git checkout {}".format(orig_branch))

    return 0

if __name__ == '__main__':
    LOG_LEVEL = logging.INFO
    if '--debug' in sys.argv[1:]:
        LOG_LEVEL = logging.DEBUG
    logging.basicConfig(level=LOG_LEVEL, format='[%(levelname)s]\t%(funcName)s:\t%(message)s')

    parser = argparse.ArgumentParser(description='Kubespector build and packaging script.')
    parser.add_argument('--verbose','-v','--debug',
                        action='store_true',
                        help='Use debug output')
    parser.add_argument('--outdir', '-o',
                        metavar='<output directory>',
                        default='./build/',
                        type=os.path.abspath,
                        help='Output directory')
    parser.add_argument('--name', '-n',
                        metavar='<name>',
                        type=str,
                        help='Name to use for package name (when package is specified)')
    parser.add_argument('--arch',
                        metavar='<amd64|i386|armhf|arm64|armel|all>',
                        type=str,
                        default=get_system_arch(),
                        help='Target architecture for build output')
    parser.add_argument('--platform',
                        metavar='<linux|darwin|windows|all>',
                        type=str,
                        default=get_system_platform(),
                        help='Target platform for build output')
    parser.add_argument('--branch',
                        metavar='<branch>',
                        type=str,
                        default=get_current_branch(),
                        help='Build from a specific branch')
    parser.add_argument('--commit',
                        metavar='<commit>',
                        type=str,
                        default=get_current_commit(short=True),
                        help='Build from a specific commit')
    parser.add_argument('--version',
                        metavar='<version>',
                        type=str,
                        default=get_current_version(),
                        help='Version information to apply to build output (ex: 0.1.0)')
    parser.add_argument('--nightly',
                        action='store_true',
                        help='Mark build output as nightly build (will incremement the minor version)')
    parser.add_argument('--release',
                        action='store_true',
                        help='Mark build output as release')
    parser.add_argument('--clean',
                        action='store_true',
                        help='Clean output directory before building')
    parser.add_argument('--no-uncommitted',
                        action='store_true',
                        help='Fail if uncommitted changes exist in the working directory')
    parser.add_argument('--skip-dep',
                        action='store_true',
                        help='Skips retrieval of build dependencies')
    parser.add_argument('--build-tags',
                        metavar='<tags>',
                        help='Optional build tags to use for compilation')
    parser.add_argument('--static',
                        action='store_true',
                        help='Create statically-compiled binary output')
    args = parser.parse_args()
    print_banner()
    sys.exit(main(args))
