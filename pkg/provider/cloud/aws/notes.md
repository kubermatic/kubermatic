* security group
* control plane role, incl. policy
* worker role, incl. policy
  * used by a worker instance profile

* read only: VPC ID, route table ID



# Resource Overview

-----------------------+--------------------------------+-------------------+--------+---------------------------------
type                   | example name                   | user configurable | tagged | if not configured
-----------------------+--------------------------------+-------------------+--------+---------------------------------
EC2 VPC                | (none)                         | yes               | no     | use default VPC
EC2 route table        | (none)                         | yes               | yes    | use VPC's default route table
EC2 security group     | kubernetes-XXXXX               | yes               | yes    | create
-----------------------+--------------------------------+-------------------+--------+---------------------------------
IAM control plane role | kubernetes-XXXXX-control-plane | yes               | no     | create
IAM worker role        | kubernetes-XXXXX-worker        | no                | no     | create
IAM worker inst. prof. | kubernetes-XXXXX               | yes               | no     | create
-----------------------+--------------------------------+-------------------+--------+---------------------------------



# Reconciling

   ## EC2 VPC

   * if not set, load default VPC ID

   ## EC2 Route Table

   * if not set, load default RT in VPC
   * tag it with cluster tag

   ## EC2 Security Group

   NB: There cannot be 2 SGs with the same name! We *must* attempt
   to find the SG by its name, otherwise we will never be able to
   reconcile.

   * if not set
     * try find based on name
     * if not found, create it and set ownership tag
     * else fix cluster spec
   * else (i.e. if ID is set)
     * check if the SG still exists
     * if not found
       * (same logic as if no ID was set at all)
     * else fix cluster spec
   * recocile permissions in SG
   * tag it with cluster tag

   ## IAM Control Plane Role

   * if not set
     * try find based on name
     * if not found, create it and set ownership tag
     * else fix cluster spec
   * else (i.e. if name is set)
     * check if the role still exists
     * if not found, create it and set ownership tag
     * else fix cluster spec
   * ensure policies are attached

   ## IAM Worker Role

   This role is ONLY created and assigned to the profile
   if we OWN the worker instance profile.

   IF we own the profile and need to care about the role, then:

   * try find based on name
   * if not found, create it and set ownership tag
   * else fix cluster spec
   * ensure policies are attached

   ## IAM Worker Instance Profile

   * if not set
     * try find based on name
     * if not found, create it and set ownership tag
     * else fix cluster spec
   * else (i.e. if name is set)
     * check if the profile still exists
     * if not found, create it and set ownership tag
     * else fix cluster spec
   * IF we own the profile:
     * ensure policies are attached
     * ensure the worker role is assigned



# Cleanup

   ## VPC

   * never do anything

   ## Route Table

   * remove cluster tag

   * if no ID set in cluster object (recovery):
   * nothing to do, because
      * we never create RT's
      * in the followup "remove all tags from all tagged resources" run we would
         remove any orphaned tag on any RT anyway

   ## Security Group

   * if ID is set:
     * if valid ID:
       * if ownerTag, then delete
       * else, just remove cluster tag
     * else:
       * find based on name
       * if found
         * if ownerTag, then delete
         * else, just remove cluster tag
       * else done
   * if no ID is set:
     * (same logic as if no valid ID was set)

   ## Control Plane Role

   * if name is set:
     * if name is valid:
       * if ownerTag, then delete
       * else done
     * else done
   * else, i.e. no name:
     * find based on name
     * if name is valid:
       * if ownerTag, then delete
       * else done
     * else done

   ## Worker Instance Profile

   * if name is set:
     * if name is valid:
       * if ownerTag
         * delete role
         * delete profile
       * else done
   * else, i.e. no name:
     * find based on name
     * if name is valid:
       * if ownerTag
         * delete role
         * delete profile
       * else done
     * else done



# Migration

In existing userclusters, we can only deduce ownership based on the
finalizers.

1. User updates KKP.
2. seed-ctrlmgr starts.
3. cloud controller begins to reconcile usercluster xyz:
   1. InitCloudProvider() does, as usual, nothing.
   2. No LastReconciled time on the cluster status, so ReconcileProvider() is forced
      1. ...
