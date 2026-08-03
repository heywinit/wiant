import { Tooltip as TooltipPrimitive } from "@base-ui/react/tooltip"
import type * as React from "react"

import { cn } from "@/lib/utils"

function TooltipProvider({ ...props }: TooltipPrimitive.Provider.Props) {
	return <TooltipPrimitive.Provider {...props} />
}

function Tooltip({
	delay = 200,
	closeDelay = 100,
	...props
}: TooltipPrimitive.Root.Props & {
	delay?: number
	closeDelay?: number
}) {
	return <TooltipPrimitive.Root {...props} />
}

function TooltipTrigger({
	className,
	render,
	...props
}: TooltipPrimitive.Trigger.Props) {
	return (
		<TooltipPrimitive.Trigger
			render={render}
			className={cn(className)}
			{...props}
		/>
	)
}

function TooltipPortal({ ...props }: TooltipPrimitive.Portal.Props) {
	return <TooltipPrimitive.Portal {...props} />
}

function TooltipContent({
	className,
	side = "top",
	sideOffset = 10,
	children,
	...props
}: TooltipPrimitive.Popup.Props &
	TooltipPrimitive.Positioner.Props & {
		children: React.ReactNode
	}) {
	return (
		<TooltipPortal>
			<TooltipPrimitive.Positioner side={side} sideOffset={sideOffset}>
				<TooltipPrimitive.Popup
					className={cn(
						"relative z-50 max-w-xs rounded-md bg-primary px-2 py-1.5 text-primary-foreground text-xs shadow-md",
						"data-ending-style:fade-out-0 data-starting-style:fade-in-0",
						className
					)}
					{...props}
				>
					{children}
					<TooltipPrimitive.Arrow className="absolute left-1/2 size-2 -translate-x-1/2 rotate-45 bg-primary data-[side=bottom]:-top-1 data-[side=left]:top-1/2 data-[side=right]:top-1/2 data-[side=left]:-right-1 data-[side=top]:-bottom-1 data-[side=left]:left-auto data-[side=right]:-left-1 data-[side=right]:translate-x-0 data-[side=left]:-translate-y-1/2 data-[side=right]:-translate-y-1/2" />
				</TooltipPrimitive.Popup>
			</TooltipPrimitive.Positioner>
		</TooltipPortal>
	)
}

export {
	Tooltip,
	TooltipContent,
	TooltipPortal,
	TooltipProvider,
	TooltipTrigger,
}
