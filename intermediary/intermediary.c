/*
 * This program is launched as a subprocess, launches the process defined on the command-line, then watches its own
 * parent for termination, killing the child and exiting.
 */
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <signal.h>

int main(int argc, char *argv[]) {
    if (argc < 2) {
        fprintf(stderr, "Usage: %s <command> [args...]\n", argv[0]);
        exit(1);
    }

    pid_t parent_pid = getppid();
    pid_t child_pid = fork();

    if (child_pid == -1) {
        perror("fork");
        exit(1);
    }

    if (child_pid == 0) {
        // Child process - execute the command
        execvp(argv[1], &argv[1]);
        perror("execvp");
        exit(1);
    }

    // Parent process - monitor original parent and wait for child
    while (1) {
        // Check if parent is still alive
        if (kill(parent_pid, 0) == -1) {
            // Parent is dead, kill child and exit
            kill(child_pid, SIGTERM);
            wait(NULL);
            exit(0);
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
