# Thyella

Thyella is software for deleting nodes belonging to a specific node pool. It is mainly used to operate the preemptible instance of GKE effectively.

## Usage

Create a container image and execute it periodically with Cronjob.

## Settings

Require environments:

```
export THYELLA_PROJECT_ID=myprojectok
export THYELLA_CLUSTER=mycluster
export THYELLA_NODE_POOLS=default-pool,preemptible-pool
```
