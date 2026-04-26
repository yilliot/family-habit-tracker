<?php

namespace App\Models;

use Database\Factories\RewardFactory;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class Reward extends Model
{
    /** @use HasFactory<RewardFactory> */
    use HasFactory;

    protected $fillable = [
        'person_id',
        'name',
        'cost',
        'achieved_at',
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
     * Validation rules for the `cost` attribute.
     *
     * @return array<int, string|int>
     */
    public static function costRules(): array
    {
        return ['required', 'integer', 'min:1', 'max:9999'];
    }

    /**
     * @return BelongsTo<Person, $this>
     */
    public function person(): BelongsTo
    {
        return $this->belongsTo(Person::class)->withTrashed();
    }

    public function isAchieved(): bool
    {
        return $this->achieved_at !== null;
    }

    /**
     * @param  Builder<Reward>  $query
     */
    public function scopeActive(Builder $query): void
    {
        $query->whereNull('achieved_at');
    }

    /**
     * @param  Builder<Reward>  $query
     */
    public function scopeAchieved(Builder $query): void
    {
        $query->whereNotNull('achieved_at');
    }

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return [
            'cost' => 'integer',
            'achieved_at' => 'datetime',
        ];
    }
}
