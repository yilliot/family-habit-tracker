<?php

namespace App\Models;

use Database\Factories\SprintParticipantFactory;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;

class SprintParticipant extends Model
{
    /** @use HasFactory<SprintParticipantFactory> */
    use HasFactory;

    protected $fillable = [
        'sprint_id',
        'person_id',
        'carry_forward_balance',
        'active_reward_id',
        'display_order',
    ];

    protected $attributes = [
        'carry_forward_balance' => 0,
        'display_order' => 0,
    ];

    /**
     * @return BelongsTo<Sprint, $this>
     */
    public function sprint(): BelongsTo
    {
        return $this->belongsTo(Sprint::class);
    }

    /**
     * @return BelongsTo<Person, $this>
     */
    public function person(): BelongsTo
    {
        return $this->belongsTo(Person::class)->withTrashed();
    }

    /**
     * @return BelongsTo<Reward, $this>
     */
    public function activeReward(): BelongsTo
    {
        return $this->belongsTo(Reward::class, 'active_reward_id');
    }

    /**
     * @return HasMany<Habit, $this>
     */
    public function habits(): HasMany
    {
        return $this->hasMany(Habit::class);
    }

    /**
     * @return HasMany<PointAdjustment, $this>
     */
    public function adjustments(): HasMany
    {
        return $this->hasMany(PointAdjustment::class);
    }

    /**
     * @param  Builder<SprintParticipant>  $query
     */
    public function scopeOrdered(Builder $query): void
    {
        $query->orderBy('display_order')->orderBy('id');
    }

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return [
            'carry_forward_balance' => 'integer',
            'display_order' => 'integer',
        ];
    }
}
