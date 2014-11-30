## Mailbox package

The mailbox package handles the data transactions and is used by all
components. It queues and dequeues jobs, as well as manages users and
their inboxes.

This is the data layer of the application and it interacts directly with 
the database.

#### Enqueuer
--
Routes messages. Inbound messages are delivered to the recipient inboxes and outbound messages are placed on the queued to be picked up by the agent.

#### Dequeuer
--
Retrieves and manages jobs from the queue.

#### Interface
--
Interface is the mailbox's interface. It contains methods for its creation, as well as for inbox mail retrieval and authentication.
