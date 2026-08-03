import type * as React from "react"
import { Input } from "@/components/ui/input"
import {
	Tooltip,
	TooltipContent,
	TooltipTrigger,
} from "@/components/ui/tooltip"

export function FormTooltipInput({
	error,
	...props
}: React.ComponentProps<typeof Input> & { error?: string }) {
	return (
		<Tooltip open={Boolean(error)}>
			<TooltipTrigger render={<div />}>
				<Input aria-invalid={Boolean(error)} {...props} />
			</TooltipTrigger>
			{error && <TooltipContent side="right">{error}</TooltipContent>}
		</Tooltip>
	)
}
