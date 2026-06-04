import * as React from "react";

interface Props {
    confirmations: number;
    requiredConfirmations: number;
    isMempool: boolean;
}

const ConfirmationProgress: React.FC<Props> = ({confirmations, requiredConfirmations, isMempool}) => {
    const circleRadius = 30;
    const circumference = 2 * Math.PI * circleRadius;
    const safeRequiredConfirmations = Math.max(requiredConfirmations, 1);
    const safeConfirmations = Math.min(Math.max(confirmations, 0), safeRequiredConfirmations);
    const progress = safeConfirmations / safeRequiredConfirmations;

    return (
        <div className="flex flex-col items-center justify-center mb-4">
            <div className="relative h-16 w-16 flex items-center justify-center">
                <svg className="transform -rotate-90 w-16 h-16">
                    <circle
                        cx="32"
                        cy="32"
                        r="30"
                        stroke="currentColor"
                        strokeWidth="4"
                        fill="transparent"
                        className="text-main-green-3"
                    />

                    <circle
                        cx="32"
                        cy="32"
                        r="30"
                        stroke="currentColor"
                        strokeWidth="4"
                        fill="transparent"
                        strokeDasharray={circumference}
                        strokeDashoffset={circumference - progress * circumference}
                        className="text-main-green-1"
                    />
                </svg>
                <span className="absolute font-medium text-sm text-main-green-1">
                    {safeConfirmations}/{safeRequiredConfirmations}
                </span>
            </div>
            <span className="mt-1 text-xs font-medium text-card-desc">{isMempool ? "In mempool" : "Confirming"}</span>
        </div>
    );
};

export default ConfirmationProgress;
