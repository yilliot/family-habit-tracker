<?php

namespace App\Models;

use Database\Factories\SprintFactory;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Illuminate\Database\Eloquent\Relations\HasManyThrough;

class Sprint extends Model
{
    /** @use HasFactory<SprintFactory> */
    use HasFactory;

    public const STATUS_ACTIVE = 'active';

    public const STATUS_ARCHIVED = 'archived';

    protected $fillable = [
        'start_date',
        'end_date',
        'status',
        'started_at',
        'ended_at',
    ];

    protected $attributes = [
        'status' => self::STATUS_ACTIVE,
    ];

    /**
     * @return HasMany<SprintParticipant, $this>
     */
    public function participants(): HasMany
    {
        return $this->hasMany(SprintParticipant::class);
    }

    /**
     * @return HasManyThrough<Habit, SprintParticipant, $this>
     */
    public function habits(): HasManyThrough
    {
        return $this->hasManyThrough(Habit::class, SprintParticipant::class);
    }

    public function isActive(): bool
    {
        return $this->status === self::STATUS_ACTIVE;
    }

    public function isArchived(): bool
    {
        return $this->status === self::STATUS_ARCHIVED;
    }

    /**
     * @param  Builder<Sprint>  $query
     */
    public function scopeActive(Builder $query): void
    {
        $query->where('status', self::STATUS_ACTIVE);
    }

    /**
     * @param  Builder<Sprint>  $query
     */
    public function scopeArchived(Builder $query): void
    {
        $query->where('status', self::STATUS_ARCHIVED);
    }

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return [
            'start_date' => 'date',
            'end_date' => 'date',
            'started_at' => 'datetime',
            'ended_at' => 'datetime',
        ];
    }
}
