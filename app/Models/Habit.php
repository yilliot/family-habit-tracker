<?php

namespace App\Models;

use Database\Factories\HabitFactory;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Illuminate\Database\Eloquent\SoftDeletes;

class Habit extends Model
{
    /** @use HasFactory<HabitFactory> */
    use HasFactory, SoftDeletes;

    protected $fillable = [
        'sprint_participant_id',
        'name',
        'display_order',
    ];

    protected $attributes = [
        'display_order' => 0,
    ];

    /**
     * Validation rules for the `name` attribute.
     *
     * @return array<int, string>
     */
    public static function nameRules(): array
    {
        return ['required', 'string', 'min:1', 'max:80'];
    }

    /**
     * @return BelongsTo<SprintParticipant, $this>
     */
    public function sprintParticipant(): BelongsTo
    {
        return $this->belongsTo(SprintParticipant::class);
    }

    /**
     * @return HasMany<Tick, $this>
     */
    public function ticks(): HasMany
    {
        return $this->hasMany(Tick::class);
    }

    /**
     * @param  Builder<Habit>  $query
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
            'display_order' => 'integer',
        ];
    }
}
