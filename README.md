_WORK IN PROGRESS_

## Full service Mail Server

__SMTP Package__ - Starts an SMTP server
__Mailbox Package__ - Data layer used by all components (queues and dequeues outbound messages, handles inboxes and other data storage interactions)
__Agent Package__ - Dequeues jobs from the mailbox queue and attempts to deliver to their recipients
__.conf files__ - Server configuration. Interpreted by [jamon](github.com/gbbr/jamon)

__(more to come)__
