import type { VariantProps } from 'class-variance-authority'
import type { ComponentProps } from 'react'
import { Slot } from '@radix-ui/react-slot'
import { cva } from 'class-variance-authority'
import { cn } from '@/shared/lib/utils'

const buttonVariants = cva(
  'inline-flex h-10 items-center justify-center gap-2 whitespace-nowrap rounded-lg px-4 py-2 text-sm font-black shadow-sm transition disabled:pointer-events-none disabled:bg-[#d9d2c5] disabled:text-[#62594d] disabled:shadow-none disabled:ring-[#191611]/8 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background',
  {
    variants: {
      variant: {
        default: 'bg-[#191611] text-[#fff8e8] shadow-[0_12px_26px_rgba(25,22,17,0.18)] hover:-translate-y-0.5 hover:bg-[#2b241c]',
        secondary: 'bg-[#fff8e8] text-[#191611] ring-1 ring-[#191611]/12 hover:-translate-y-0.5 hover:bg-white',
        outline: 'bg-[#fff8e8] text-[#191611] ring-1 ring-[#191611]/16 hover:-translate-y-0.5 hover:bg-white',
        ghost: 'bg-transparent text-current shadow-none hover:bg-[#191611]/8',
        destructive: 'bg-[#c43d28] text-[#fff8e8] hover:-translate-y-0.5 hover:bg-[#a92f1e]',
      },
      size: {
        default: 'h-10 px-4 py-2',
        sm: 'h-9 rounded-lg px-3',
        lg: 'h-11 rounded-lg px-8',
        icon: 'size-10',
      },
    },
    defaultVariants: {
      variant: 'default',
      size: 'default',
    },
  },
)

type ButtonProps = ComponentProps<'button'>
  & VariantProps<typeof buttonVariants> & {
    asChild?: boolean
  }

export function Button({
  className,
  variant,
  size,
  asChild = false,
  ...props
}: ButtonProps) {
  const Comp = asChild ? Slot : 'button'

  return (
    <Comp
      className={cn(buttonVariants({ variant, size, className }))}
      {...props}
    />
  )
}
