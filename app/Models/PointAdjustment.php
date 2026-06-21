<?php

namespace App\Models;

use Database\Factories\PointAdjustmentFactory;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class PointAdjustment extends Model
{
    /** @use HasFactory<PointAdjustmentFactory> */
    use HasFactory;

    public $timestamps = false;

    protected $fillable = [
        'sprint_participant_id',
        'amount',
        'reason',
        'created_at',
    ];

    /**
     * @return BelongsTo<SprintParticipant, $this>
     */
    public function sprintParticipant(): BelongsTo
    {
        return $this->belongsTo(SprintParticipant::class);
    }

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return [
            'amount' => 'integer',
            'created_at' => 'datetime',
        ];
    }
}
