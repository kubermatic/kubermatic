# seedresourcesuptodatecondition controller

Purpose:
	* Setting a condition on the cluster object when all Deployments and Statefulsets were fully rolled out
	* This condition serves the purpose of limiting the load imposed on the seed
	* All controllers that create Deployments or Statefulsets in the seed must respect it
		via `controllerutil.ClusterAvailableForReconciling`
