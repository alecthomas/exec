/*
 * This program is launched as a subprocess, launches the process defined on the command-line, then watches its own
 * parent for termination, killing the child and exiting.
 */
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <signal.h>
#include <assert.h>

static pid_t child_pid = -1;

void cleanup_and_exit(int exit_code, int signal) {
  if (child_pid > 0) {
    kill(-child_pid, signal);
    wait(NULL);
  }
  exit(exit_code);
}

void signal_handler(int sig) {
  if (child_pid == -1) {
    return;
  }
  cleanup_and_exit(0, sig);
}

int main(int argc, char *argv[]) {
  if (argc < 2) {
    fprintf(stderr, "Usage: %s <command> [args...]\n", argv[0]);
    exit(1);
  }

  pid_t parent_pid = getppid();

  child_pid = fork();

  if (child_pid == -1) {
    perror("fork");
    exit(1);
  }

  if (child_pid == 0) {
    // Child process - create new process group and execute the command
    if (setpgid(0, 0) == -1) {
      perror("setpgid");
      exit(1);
    }
    execvp(argv[1], &argv[1]);
    perror("execvp");
    exit(1);
  }

  // Set up signal handlers before forking
  signal(SIGTERM, signal_handler);
  signal(SIGINT, signal_handler);
  signal(SIGHUP, signal_handler);

  // Parent process - monitor original parent and wait for child
  while (1) {
    // Check if parent is still alive
    if (kill(parent_pid, 0) == -1) {
      // Parent is dead, kill child process group and exit
      cleanup_and_exit(0, SIGTERM);
    }

    // Check if child has exited
    int status;
    pid_t result = waitpid(child_pid, &status, WNOHANG);
    if (result == child_pid) {
      // Child has exited
      exit(WIFEXITED(status) ? WEXITSTATUS(status) : 1);
    }

    // Sleep briefly to avoid busy waiting
    usleep(100000); // 100ms
  }
}
