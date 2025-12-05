#include <signal.h>
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <string.h>

static volatile int ack_received = 0;
static int current_bit_index = 0;
static unsigned int number_to_send = 0;
static pid_t receiver_pid = 0;

void ack_handler(int sig) {
    ack_received = 1;
}

void send_next_bit(void) {
    if (current_bit_index < 32) {
        int bit = (number_to_send >> (31 - current_bit_index)) & 1;

        if (bit == 0) {
            kill(receiver_pid, SIGUSR1);
        } else {
            kill(receiver_pid, SIGUSR2);
        }

        current_bit_index++;
    }
}

int main(void) {
    int number;
    int i;
    char binary[33];

    memset(binary, 0, sizeof(binary));

    printf("Sender pid = %d\n", getpid());
    printf("Input receiver PID: ");
    if (scanf("%d", &receiver_pid) != 1) {
        fprintf(stderr, "Invalid PID\n");
        return 1;
    }

    printf("Input decimal integer number: ");
    if (scanf("%d", &number) != 1) {
        fprintf(stderr, "Invalid number\n");
        return 1;
    }

    number_to_send = *(unsigned int *)&number;

    for (i = 31; i >= 0; i--) {
        binary[31 - i] = ((number_to_send >> i) & 1) ? '1' : '0';
    }
    binary[32] = '\0';

    printf("%s\n", binary);
    fflush(stdout);

    signal(SIGUSR1, ack_handler);

    current_bit_index = 0;

    send_next_bit();

    while (current_bit_index < 32) {
        if (ack_received) {
            ack_received = 0;
            send_next_bit();
        }
        usleep(10);
    }

    return 0;
}
