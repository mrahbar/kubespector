import os, tarfile, argparse
import shlex
import subprocess
import xml.etree.ElementTree as ET
import time
import sys

SNAPSHOT_VERSION_SUFFIX = "SNAPSHOT"
MVN_PATH_SETTINGS_XML = 'D:\\Users\\maaz1de\\.m2\\settings_prod.xml'
STEP_COUNT = 3


class TerminalColors:
    HEADER = '\033[95m'
    OKBLUE = '\033[94m'
    OKGREEN = '\033[92m'
    WARNING = '\033[93m'
    FAIL = '\033[91m'
    BOLD = '\033[1m'
    UNDERLINE = '\033[4m'
    ENDC = '\033[0m'


def print_info(msg):
    print TerminalColors.OKBLUE + msg + TerminalColors.ENDC


def print_success(msg):
    print TerminalColors.OKGREEN + msg + TerminalColors.ENDC


def print_warning(msg):
    print TerminalColors.WARNING + msg + TerminalColors.ENDC


def print_fail(msg):
    print TerminalColors.FAIL + msg + TerminalColors.ENDC


def print_underscore(msg):
    print TerminalColors.UNDERLINE + msg + TerminalColors.ENDC


def print_bold(msg):
    print TerminalColors.BOLD + msg + TerminalColors.ENDC


def print_header(msg):
    print TerminalColors.HEADER + msg + TerminalColors.ENDC


def log_timing(step_id, step_count, start_time):
    step_id = str(step_id)
    print_underscore("\tTiming: Step %s/%s took %0.1f seconds" % (step_id, step_count, time.time() - start_time))


class Deployer(object):
    def __init__(self):
        super(Deployer, self).__init__()
        self.step_counter = 1

    def run(self, args):
        print_underscore("")
        print_header("Starting task deploy")

        if not args.repo_type in ['snapshots', 'releases']:
            print_fail("Deployment failed unknown repo-type: %s" % args.repo_type)
            self.finish_and_exit()

        # Build nexus path
        start_time = time.time()
        nexus_path = self.build_nexus_path(args.repo_type)
        log_timing(self.step_counter, STEP_COUNT, start_time)
        self.step_counter += 1

        # Package dist
        artifact_name = self.package_dist(args)
        self.step_counter += 1

        # Perform artifact upload
        return_code, result = self.upload_artifact(args, artifact_name, nexus_path, args.repo_type)
        self.step_counter += 1

        if return_code == 0:
            print_success("Deployment successful")
            self.finish_task()
        else:
            print_fail("Deployment failed with return code %s Stdout: \'%s\' Stderr: \'%s\'" %
                       (str(return_code), self.get_stdout(result), self.get_stderr(result)))
            self.finish_and_exit()

    def package_dist(self, args):
        start_time = time.time()
        output_name = "%s-%s" % (args.artifact_name, args.artifact_version)
        output_filename = "%s.tar.gz" % output_name
        print_info("Packaging artifact %s" % output_filename)

        with tarfile.open(os.path.join(os.getcwd(), output_filename), "w:gz") as tar:
            for r in args.project_resources:
                tar.add(r)

        log_timing(self.step_counter, STEP_COUNT, start_time)

        return output_filename

    def read_nexus_credentials_from_settings(self):
        if os.path.isfile(MVN_PATH_SETTINGS_XML):
            tree = ET.parse('%s' % MVN_PATH_SETTINGS_XML)
            username_tags = tree.findall(".//username")
            password_tags = tree.findall(".//password")

            if len(username_tags) > 0 and len(password_tags) > 0:
                username = username_tags[0].text
                password = password_tags[0].text

                return username, password
            else:
                print_fail("Could not read nexus credentials from settings.")
                self.finish_and_exit()
        else:
            print_fail("File settings.xml does not exists under expected root: %s" % MVN_PATH_SETTINGS_XML)
            self.finish_and_exit()

    def build_nexus_path(self, repo_type):
        nexus_user, nexus_password = self.read_nexus_credentials_from_settings()

        nexus_path = "http://%(nexus_user)s:%(nexus_password)s@mucsgpop01.sg.de.pri.o2.com:8080/nexus/content/repositories/%(nexus_repo_type)s" % {
            'nexus_user': nexus_user,
            'nexus_password': nexus_password,
            'nexus_repo_type': repo_type}
        return nexus_path

    def upload_artifact(self, args, artifact_name, nexus_path, repo_type):
        start_time = time.time()
        print_info("Uploading artifact to repo \'%s\'" % repo_type)
        return_code, result = self.call_command("mvn deploy:deploy-file " \
                                                "-DgroupId=%(groupId)s " \
                                                "-DartifactId=%(current_repo)s " \
                                                "-Dversion=%(nexus_version)s " \
                                                "-DgeneratePom=true " \
                                                "-Dpackaging=tar.gz " \
                                                "-Durl=%(nexus_path)s " \
                                                "-Dfile=%(artifact_name)s " \
                                                "-l %(nexus_logfile)s" % {
                                                    'groupId': args.groupId,
                                                    'current_repo': args.artifact_name,
                                                    'nexus_version': args.artifact_version,
                                                    'nexus_path': nexus_path,
                                                    'artifact_name': artifact_name,
                                                    'nexus_logfile': './nexus.log'})

        log_timing(self.step_counter, STEP_COUNT, start_time)
        return return_code, result

    def get_stdout(self, result):
        return result[1]

    def get_stderr(self, result):
        return result[0]

    def call_command(self, command, debug=False):
        args = shlex.split(command)

        if debug:
            print args

        env = os.environ
        shell = sys.platform == 'win32'

        process = subprocess.Popen(args, shell=shell, env=env, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        result = process.communicate()
        return_code = process.returncode

        if debug:
            print "Stdout: \'%s\' Stderr: \'%s\'" % (self.get_stdout(result), self.get_stderr(result))

        return return_code, result

    def finish_and_exit(self):
        self.finish_task()
        sys.exit(-1)

    def finish_task(self):
        print_info("Deployment task finished.")
        print_underscore("")


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description="Build script for Angular 2 applications."
                                                 " This script is intended to be run at the location of \'package.json\'."
                                                 " It uses \'npm\' to call npm run <script name>.")

    parser.add_argument('--artifact-name', '-a', required=True, help='Name of artifact', dest="artifact_name", type=str)
    parser.add_argument('--artifact-version', '-v', required=True, help='Version of artifact', dest="artifact_version",
                        type=str)
    parser.add_argument('--group-id', '-g', required=True, help='GroupId of artifact', dest="groupId", type=str)
    parser.add_argument('--project-resources', '-p', required=True, dest="project_resources",
                        help='File and folder name of project resources which should be added to the bundled artifact', type=str,
                        nargs='+')
    parser.add_argument('--repo-type', '-r', required=True, help='Type of repository: snapshots or releases',
                        dest="repo_type", type=str)
    parser.add_argument('--debug', action='store_true', default=False, required=False,
                        help='Enable output of call commands.')

    args = parser.parse_args()
    Deployer().run(args)