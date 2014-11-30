## Mailbox package

The mailbox package handles the data transactions and is used by all
components. It queues and dequeues jobs, as well as manages users and
their inboxes.

This is the data layer of the application and it interacts directly with 
the database.  

--

### Components

__Enqueuer__  
Routes messages. Inbound messages are delivered to the recipient inboxes and outbound messages are placed on the queue to be picked up by the agent. This interface is used by the SMTP server.

__Dequeuer__  
Retrieves and manages jobs from the queue. This interface is used by the mail delivery agent.

__Interface__  
Interface is the mailbox's interface. It contains methods for its creation, as well as for inbox mail retrieval and authentication. This interface is used by the POP3 server.
