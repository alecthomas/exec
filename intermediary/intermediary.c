#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/wait.h>
#include <signal.h>
#include <errno.h>
#include <string.h>

// Debug mode - set to 1 to enable debug logging
#define DEBUG_MODE 0

#if DEBUG_MODE
#define debug_log(fmt, ...) do { \
    fprintf(stderr, "[PID:%d] " fmt "\n", getpid(), ##__VA_ARGS__); \
    fflush(stderr); \
} while(0)
#else
#define debug_log(fmt, ...) do { } while(0)
#endif

// Function to create a new process group
static int create_process_group(void) {
    debug_log("Creating process group");
    if (setpgrp() == -1) {
        perror("setpgrp");
        return -1;
    }
    debug_log("Process group created: PGID=%d", getpgrp());
    return 0;
}

// Function to kill the entire process group
static void kill_process_group(void) {
    pid_t pgid = getpgrp();
    debug_log("Killing process group PGID=%d", pgid);
    
    // Don't kill our own process group if we're the session leader
    if (pgid == getpid()) {
        debug_log("Skipping - we are session leader");
        return;
    }
    
    // First try SIGTERM
    if (killpg(pgid, SIGTERM) == -1) {
        if (errno != ESRCH) {
            perror("killpg SIGTERM");
        }
    } else {
        debug_log("Sent SIGTERM to process group");
    }
    
    // Give processes time to terminate gracefully
    usleep(100000); // 100ms
    
    // Force kill with SIGKILL
    if (killpg(pgid, SIGKILL) == -1) {
        if (errno != ESRCH) {
            perror("killpg SIGKILL");
        }
    } else {
        debug_log("Sent SIGKILL to process group");
    }
}

// Function to check if parent process is still alive
static int is_parent_alive(pid_t parent_pid) {
    return kill(parent_pid, 0) == 0;
}

// Function to execute the target program
static void exec_program(int argc, char *argv[]) {
    if (argc < 2) {
        fprintf(stderr, "Usage: %s <program> [args...]\n", argv[0]);
        exit(1);
    }
    
    debug_log("Executing: %s", argv[1]);
    execvp(argv[1], &argv[1]);
    perror("execvp");
    exit(1);
}

// Watchdog function that monitors parent and child processes
static int run_watchdog(pid_t parent_pid, pid_t child_pid) {
    debug_log("Starting watchdog: parent=%d child=%d", parent_pid, child_pid);
    
    while (1) {
        int status;
        pid_t result;
        
        // Check if parent is still alive
        if (!is_parent_alive(parent_pid)) {
            debug_log("Parent %d died, killing process group", parent_pid);
            kill_process_group();
            exit(1);
        }
        
        // Check if our specific child has exited
        result = waitpid(child_pid, &status, WNOHANG);
        if (result == child_pid) {
            debug_log("Child %d exited", child_pid);
            if (WIFEXITED(status)) {
                int exit_code = WEXITSTATUS(status);
                debug_log("Child exited with code %d", exit_code);
                exit(exit_code);
            } else if (WIFSIGNALED(status)) {
                int exit_code = 128 + WTERMSIG(status);
                debug_log("Child killed by signal %d", WTERMSIG(status));
                exit(exit_code);
            } else {
                debug_log("Child exited with unknown status");
                exit(1);
            }
        } else if (result == -1 && errno == ECHILD) {
            debug_log("Child no longer exists");
            exit(0);
        } else if (result == -1) {
            perror("waitpid");
            exit(1);
        }
        
        // Reap any other children to prevent zombies
        while ((result = waitpid(-1, &status, WNOHANG)) > 0) {
            if (result == child_pid) {
                debug_log("Reaped monitored child %d", result);
                if (WIFEXITED(status)) {
                    exit(WEXITSTATUS(status));
                } else if (WIFSIGNALED(status)) {
                    exit(128 + WTERMSIG(status));
                } else {
                    exit(1);
                }
            } else {
                debug_log("Reaped unexpected child %d", result);
            }
        }
        
        // Short sleep to avoid busy waiting
        usleep(50000); // 50ms
    }
}

// Second fork - creates the final child that will exec the program
static int second_fork(pid_t parent_to_watch, int argc, char *argv[]) {
    debug_log("Performing second fork");
    pid_t child_pid = fork();
    
    if (child_pid == -1) {
        perror("second fork");
        return -1;
    }
    
    if (child_pid == 0) {
        // Final child process - exec the target program
        debug_log("Final child about to exec");
        exec_program(argc, argv);
        return -1; // Never reached
    } else {
        // Intermediate process becomes watchdog
        debug_log("Intermediate watchdog for child %d", child_pid);
        return run_watchdog(parent_to_watch, child_pid);
    }
}

// First fork - creates the intermediate watchdog process
static int first_fork(pid_t original_parent, int argc, char *argv[]) {
    debug_log("Performing first fork");
    pid_t current_pid = getpid();
    pid_t child_pid = fork();
    
    if (child_pid == -1) {
        perror("first fork");
        return -1;
    }
    
    if (child_pid == 0) {
        // First child - will become intermediate watchdog
        debug_log("First child will become intermediate");
        return second_fork(current_pid, argc, argv);
    } else {
        // Original process becomes watchdog for first child
        debug_log("Original watchdog for child %d", child_pid);
        return run_watchdog(original_parent, child_pid);
    }
}

int main(int argc, char *argv[]) {
    pid_t original_parent = getppid();
    
    debug_log("Intermediary starting: PID=%d PPID=%d", getpid(), original_parent);
    
    // Create a new process group
    if (create_process_group() == -1) {
        return 1;
    }
    
    // Start the double fork chain
    return first_fork(original_parent, argc, argv);
}