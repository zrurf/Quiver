class_name NodeUitl

static func disable_node(node:Node):
	if is_instance_valid(node):
		node.set_process(false)
		node.set_process_internal(false)
		node.set_physics_process(false)
		node.set_physics_process_internal(false)
		node.set_process_input(false)
		node.set_process_unhandled_input(false)
		node.set_process_unhandled_key_input(false)
		if node.has_method("hide"):
			node.hide()
	else:
		printerr("the node you are trying to disable is not valid")

static func enable_node(node:Node):
	if is_instance_valid(node):
		node.set_process(true)
		node.set_process_internal(true)
		node.set_physics_process(true)
		node.set_physics_process_internal(true)
		node.set_process_input(true)
		node.set_process_unhandled_input(true)
		node.set_process_unhandled_key_input(true)
		if node.has_method("show"):
			node.show()
	else:
		printerr("the node you are trying to enable is not valid")
