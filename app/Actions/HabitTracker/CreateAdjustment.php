<?php

namespace App\Actions\HabitTracker;

use App\Models\PointAdjustment;
use App\Models\SprintParticipant;
use Illuminate\Validation\ValidationException;

class CreateAdjustment
{
    /**
     * Record a manual signed point adjustment for a participant. A negative
     * amount is a deduction; a positive amount is a manual award. Balances
     * are allowed to go negative, so no balance floor is enforced here.
     */
    public function execute(SprintParticipant $participant, int $amount, ?string $reason = null): PointAdjustment
    {
        $participant->loadMissing('sprint');

        if ($participant->sprint->isArchived()) {
            throw ValidationException::withMessages([
                'sprint' => 'Cannot adjust points on an archived sprint.',
            ]);
        }

        if ($amount === 0) {
            throw ValidationException::withMessages([
                'amount' => 'Adjustment amount cannot be zero.',
            ]);
        }

        $trimmedReason = $reason !== null ? trim($reason) : null;

        return $participant->adjustments()->create([
            'amount' => $amount,
            'reason' => $trimmedReason === '' ? null : $trimmedReason,
            'created_at' => now(),
        ]);
    }
}
