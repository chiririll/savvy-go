import type { SVGProps } from 'react'

export function Logo({ className = 'size-6' }: LogoProps) {
    return (
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100" fill="none" {...props}>
            {/* Speed lines */}
            <g stroke="#9ca3af" strokeWidth="6" strokeLinecap="round">
                <path d="M16,38 L26,38" />
                <path d="M6,50 L30,50" />
                <path d="M16,62 L34,62" />
            </g>
            {/* "VV" mark */}
            <g
                transform="translate(22 10) scale(0.8)"
                stroke="currentColor"
                strokeWidth="8"
                strokeLinecap="round"
                strokeLinejoin="round"
            >
                <path d="M15,28 L30,72 L45,28" />
                <path d="M55,72 L70,28 L85,72" />
            </g>
        </svg>
    )
}
