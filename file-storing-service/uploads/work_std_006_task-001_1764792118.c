#include <signal.h>
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <string.h>

static unsigned int received_number = 0;
static int bit_count = 0;
static char binary[33];
static pid_t sender_pid = 0;

void sigusr1_handler(int sig) {
    binary[bit_count] = '0';
    bit_count++;

    kill(sender_pid, SIGUSR1);
}

void sigusr2_handler(int sig) {
    received_number |= (1u << (31 - bit_count));
    binary[bit_count] = '1';
    bit_count++;

    kill(sender_pid, SIGUSR1);
}

int main(void) {
    struct sigaction sa1, sa2;
    int dummy;

    memset(binary, 0, sizeof(binary));

    printf("Receiver pid = %d\n", getpid());
    printf("Input sender PID: ");
    if (scanf("%d", &sender_pid) != 1) {
        fprintf(stderr, "Invalid PID\n");
        return 1;
    }

    memset(&sa1, 0, sizeof(sa1));
    sa1.sa_handler = sigusr1_handler;
    sigaction(SIGUSR1, &sa1, NULL);

    memset(&sa2, 0, sizeof(sa2));
    sa2.sa_handler = sigusr2_handler;
    sigaction(SIGUSR2, &sa2, NULL);

    while (bit_count < 32) {
        pause();
    }

    binary[32] = '\0';
    printf("%s\n", binary);

    int result = *(int *)&received_number;
    printf("Result = %d\n", result);
    fflush(stdout);

    return 0;
}
